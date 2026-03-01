# Archived CSV Removal — What Was NOT Changed (and Why)

These notes document functions and config that were intentionally left unchanged when archived CSV logic was removed from the main pipeline. Captured during the planning phase for reference.

---

## `MarkDisappearedInactive` — stays, behavior changes slightly

**File:** `pipeline/internal/database/opportunities.go:247-261`

Sets `active = false` on opportunities that have an unresolved disappearance (`resolution IS NULL`) for the given `run_id`. Joins `snap_disappearances` to `opportunities` on `notice_id` and only affects rows where `is_latest = true`.

**Called from:** `pipeline/cmd/lambda/reconcile/main.go:271` — always runs when `activeRunID != uuid.Nil`.

**Before this change:** `ResolveExpectedDisappearances` ran first, marking disappearances found in the archived CSV as `resolution = 'archived'`. Then `MarkDisappearedInactive` only deactivated the remaining unresolved ones (true unexpected disappearances).

**After this change:** Without archived resolution, every disappearance stays `resolution IS NULL`, so `MarkDisappearedInactive` deactivates all of them. This is correct — if an opportunity vanishes from the active CSV and we have no archived data to explain why, we mark it inactive.

---

## `DetectDisappearances` — active-only, unchanged

**File:** `pipeline/internal/database/snapshot.go:136-178`

Compares two active CSV runs (`currentRunID` vs `previousRunID`). For each `notice_id` present in the previous run but missing from the current run, inserts a row into `pipeline.snap_disappearances` with metadata (`last_seen_date`, `last_type`, `last_archive_type`, etc.). Uses batch inserts.

**Called from:** `pipeline/cmd/lambda/ingest-active/main.go:308` — only from the ingest-active Lambda, not from reconcile or ingest-archived. Takes the current and previous active run IDs.

**Why unchanged:** This function only ever compared active-vs-active runs. The archived CSV was never involved in *detecting* disappearances — it was only used later (in reconcile) to *resolve* them.

---

## `DetectReappearances` — active-only, unchanged

**File:** `pipeline/internal/database/snapshot.go:180-196`

Marks unresolved disappearances as `resolution = 'glitch'` when the `notice_id` reappears in a new active CSV run. Sets `resolution_source = 'active_csv_reappeared'` and `reappeared_date`.

**Called from:** `pipeline/cmd/lambda/ingest-active/main.go:315` — only from ingest-active.

**Why unchanged:** Same as above — this is active-vs-active logic. It resolves disappearances that turn out to be transient (notice dropped from one run but came back in the next). No archived CSV involvement.

---

## `S3ArchiveEnabled` / `archiveFile()` — NOT archived-CSV-specific

**File:** `pipeline/internal/bulkcsv/bulkcsv.go:221-316` (archiveFile), `bulkcsv.go:81,249` (S3ArchiveEnabled checks)

`archiveFile()` compresses the downloaded CSV to gzip and uploads it to S3 under `{src.S3Prefix}/{date}.csv.gz`. It logs the result to `pipeline.bulk_csv_log`. The `cfg.S3ArchiveEnabled` flag gates the S3 upload — when false, the file is still compressed and logged but not uploaded.

Despite the name "archive", this function runs for **every source** — active and archived alike. It's called from both `downloadWithETag` (line 176) and `downloadWithRangeProbe` (line 217). After removing archived sources from `bulkcsv/sources.go`, this function continues to work for the remaining active source.

**Why unchanged:** The word "archive" here means "store a copy in S3 for audit/backup", not "handle archived CSV data". Renaming it would be a nice-to-have but out of scope.

---

## `config.Config.S3ArchiveEnabled` — leave as-is

**File:** `pipeline/internal/config/config.go:17`

A boolean config flag (`S3_ARCHIVE_ENABLED` env var, defaults to `true`). Controls whether `archiveFile()` uploads to S3. Used by the `download-csvs` Lambda.

**Why unchanged:** Same reason as above — this gates S3 upload for all sources, not just archived ones.

---

## `UpsertOpportunitiesFromAPI` — dead code, clean up separately

**File:** `pipeline/internal/database/opportunities_api.go:15-395`

A ~380-line function that was the old ingest-api's way of upserting API opportunities directly into the `opportunities` table. It did COALESCE-merge with existing rows, version bumping, and content-hash comparison. This was replaced when reconciliation was moved to the reconcile Lambda (previous refactor). Now `ingest-api` only does `BulkInsertSnapAPI`, and the reconcile Lambda handles the upsert via `UpsertOpportunities`.

**Still referenced in:**
- `pipeline/internal/database/store.go` — listed in the `Store` interface
- `pipeline/internal/testutil/mock_store.go` — has a `UpsertOpportunitiesFromAPIFn` field
- `pipeline/internal/e2e/api_enrichment_test.go` — the E2E test still calls it

**Why not removed now:** It's dead in production (no Lambda calls it) but the E2E test still exercises it. Removing it requires updating E2E tests which is a separate cleanup task. Including it in this change would make the diff larger without being related to archived CSV removal.
