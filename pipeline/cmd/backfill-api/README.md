# backfill-api — one-time API-only backfill

Recovers SAM.gov opportunities for a posted-date window that the **daily CSV
pipeline missed** (e.g. during a SAM.gov CSV outage). Run it manually, once per
day, until the CSV feed is fixed on SAM.gov's side.

## Why this exists

`ingest-api`'s `FetchAll` queries **yearly** windows and pages blindly. SAM.gov
caps any single query at **1000 retrievable results** (`offset >= 1000` returns
empty — undocumented but verified), so every window over 1000 records is
**silently truncated to the first 1000**. That's why the API path never
backfilled the outage window on its own.

This command sidesteps the cap by slicing each **day × procurement type
(`ptype`)**. Those slices are exhaustive and non-overlapping (their totals sum to
the day total), and for the outage window each slice is well under 1000. If a
slice is ever ≥1000 the command **aborts loudly** rather than truncating.

## Safety

- **No deactivation.** The upsert runs via reconcile with an **API result only**,
  so `activeRunID` is nil → the disappearance/mass-deactivation path is skipped
  entirely. It can only add/refresh records, never archive the ~28K CSV-only ones.
- **Idempotent.** Writes to append-only `pipeline.snap_api`; reconcile upserts by
  `notice_id`. Re-running is safe. When the CSV feed recovers, the normal daily
  pipeline re-merges these records back to `csv+api` (self-heals).
- **Don't backfill before the last good CSV day.** Records already ingested from
  CSV are richer (`csv+api`); an API-only pass would temporarily downgrade them.
  The last healthy CSV ingestion was **2026-07-28**, so the window starts
  **07/29/2026**.
- **Compliance 🟢.** Official REST API, ~72 requests/run against a 1,000/day cap,
  1s pacing. This is the sanctioned programmatic path, not scraping.

## Prerequisites

Two environment variables (never commit these):

```bash
export NEON_DATABASE_URL="$(grep '^NEON_DATABASE_URL=' ../../.env | cut -d= -f2-)"
export SAM_API_KEY="$(aws secretsmanager get-secret-value --secret-id govtrove/sam-api-key --profile govtrove --query SecretString --output text)"
```

`SAM_API_KEY` **must** be the live key from Secrets Manager (`.env`'s copy is
stale after the rotation). Requests are logged to `pipeline.samgov_requests`
under the live key hash automatically.

## Run it

From `pipeline/`:

```bash
# 1. DRY RUN (default) — fetches and reports counts, writes nothing.
go run ./cmd/backfill-api -from 07/29/2026 -to 08/05/2026

# 2. WRITE — after the dry run looks sane, ingest into snap_api.
go run ./cmd/backfill-api -from 07/29/2026 -to 08/05/2026 -write
```

`-to` defaults to today, so for the daily re-run you can just widen it:
`-from 07/29/2026` (leave `-to` off to mean "through today").

The `-write` run prints a ready-to-paste reconcile command. Run it to upsert the
snapshot into the catalog. **Use an async invoke** (`--invocation-type Event`):
reconcile takes 1–3 min, and a synchronous invoke can exceed the AWS CLI's 60s
read timeout and get **retried**, running reconcile concurrently (deadlocks on the
expired-deactivation UPDATE — harmless but messy). Async runs it exactly once.

```bash
aws lambda invoke --function-name govtrove-reconcile --profile govtrove --invocation-type Event \
  --cli-binary-format raw-in-base64-out \
  --payload '{"execution_id":"<ANY_UUID>","api_result":{"status":"ok","run_id":"<RUN_ID>","job_type":"snapshot-api"}}' /dev/stdout
```

The `execution_id` (any fresh UUID — the command prints one) makes reconcile
record a completed `reconcile` pipeline step. `/api/status` reads that step's
`completed_at` as `last_synced_at`, so including it **advances the freshness
marker and clears the "data hasn't updated" banner**. Omit it and the data still
lands, but the banner stays stale.

It returns `202` immediately; reconcile finishes in the background (~2–3 min).
Then verify with the queries below. (Note: API-only reconcile logs many harmless
`snap_reconcile_dq ... foreign key` WARNings — it's the DQ audit skipping rows for
a run with no CSV side; the catalog upsert is unaffected.)

## Verify

```sql
-- active catalog count before/after
SELECT count(*) FILTER (WHERE active) FROM opportunities WHERE is_latest = true;
-- this run's snapshot
SELECT records_fetched, records_inserted, status FROM pipeline.ingestion_runs
  WHERE job_type = 'snapshot-api' ORDER BY started_at DESC LIMIT 1;
```

## Daily re-run checklist (until CSV is fixed)

1. `export` the two env vars (above).
2. Dry run — confirm counts look reasonable and no "SLICE TOO LARGE" abort.
3. `-write`.
4. Invoke `govtrove-reconcile` with the printed `run_id`.
5. Stop once SAM.gov's CSV feed is healthy again (a normal `snapshot-csv`
   ingestion with ~78K records reappears in `pipeline.ingestion_runs`).

## Not a replacement for CSV

The API structurally omits ~40% of the catalog (Modifications, J&As, most
standalone Solicitations — they 404 in the API). This backfill is a **majority
recovery / outage stopgap only.** See `docs/csv-vs-api-analysis.md`.
