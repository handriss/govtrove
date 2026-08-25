# Search Analytics, Signup Conversion & Search Coaching — Research and Implementation Plan

**Status:** Research complete. §7 items 1-4 are done — see §10.
**Audited 2026-08-25** against production; three claims were wrong and are corrected inline.
**Prepared:** 2026-08-24, from a read-only session against the production database.
**Intended use:** hand this to a local Claude Code session that has the full `.env` available.

Everything below was verified against production data or the code as it stands on `main`.
Line numbers are from the state of the repo at the time of writing — re-check before editing.

---

## 0. What the implementing session needs

This work cannot be done from a Claude Code web session. It needs:

- `DATABASE_URL` reachable over the Postgres wire protocol (port 5432). Web sessions block
  outbound TCP on 5432, so all research below was done via Neon's HTTPS SQL endpoint.
- `OPENROUTER_API_KEY` — the LLM tier already used for search rescue.
- The frontend toolchain for the React changes.

**Note on `DATABASE_URL`:** it currently carries a `charset=utf8` query parameter. That is not a
libpq parameter — `psql` rejects it outright (`invalid URI query parameter: "charset"`), and pgx
passes it through as a startup runtime parameter, which Postgres rejects
(`unrecognized configuration parameter "charset"`). Remove `&charset=utf8`. Keep
`sslmode=require&channel_binding=require`.

---

## 1. Research findings

### 1.1 Data hygiene — clean this up first

**A smoke test run on 2026-08-23 polluted `search_events`.** It generated roughly 1,450 searches,
114 of them zero-result — which is **24% of all zero-result searches in the table's history**.
Its queries are recognisable as mid-sentence fragments of opportunity descriptions
(e.g. `"e use the drop down box to select the most current version"`).

Purge before computing any baseline:

```sql
-- inspect first
SELECT count(*) FROM search_events
WHERE user_agent = 'govtrove-smoke-test/1.0';

DELETE FROM search_events
WHERE user_agent = 'govtrove-smoke-test/1.0';
```

All numbers in this document already exclude that window (`created_at::date < '2026-08-23'`).

**Incomplete — corrected 2026-08-25.** The date filter excludes the smoke-test window, but
the baseline still contained **368 `GovTroveE2E` events** (plus 111 from an AI crawler),
which this section never mentions. Excluding them: 2,429 -> 2,060 searches and 363 -> 309
zero-result. The *rates* are unaffected (15.0% vs 14.9%), so §1.2's headline holds, but
absolute daily volumes were inflated ~15%. Both smoke-test and E2E traffic are now filtered
out of the admin displays in code (`excludeTestTraffic`, `api/internal/repository/analytics.go`);
the rows remain in the table by choice.

### 1.2 Search outcome distribution

Over 2,452 genuine searches (2026-05-26 → 2026-08-22):

| Outcome | Count | Share |
|---|---|---|
| Zero results | 363 | **14.8%** |
| 1–5 results | 334 | **13.6%** |
| More than 5 | ~1,755 | 71.6% |

So roughly **28% of searches disappoint**. Daily averages: **4.2 zero-result** and
**3.9 few-result** searches per active day, across 86 active days. 152 distinct zero-result queries.

That volume matters for the coaching cron in §5: it is small enough that per-search LLM
evaluation is trivially affordable.

### 1.3 Engagement vs. signup

| Metric | Value |
|---|---|
| Anonymous searches | 3,527 |
| Anonymous opportunity **views** | **1,063** |
| `save_opportunity` events | 35 (all authenticated) |
| `save_search` events | 2 logged (logging began 2026-08-06 — see note) |
| `saved_searches` rows | **33, by 10 users** |
| Total users | 59 |

Two conclusions:

1. **Anonymous users engage well beyond searching** — 1,063 opportunity detail views is a much
   stronger intent signal than a failed search, and nothing currently converts on it.
2. **Saved searches are under-used** — but the "2 uses ever" figure is an instrumentation
   artifact, not user behaviour. `save_search` event logging began 2026-08-06 while
   `save_opportunity` began 2026-06-15, so the two counts cover different windows. The
   tables tell the real story: **33 saved searches by 10 users** since February, against
   66 saved opportunities by 14 users. The conclusion survives — only **1** saved search
   in the last 60 days, most recent 2026-08-17 — but it is stagnation, not near-zero
   adoption. The alerting infrastructure already exists
   (`pipeline/cmd/lambda/generate-alerts`).

Monthly signups (Feb→Aug): 3, 12, 19, 9, 6, 6, 4. **Do not read a trend into this.** The
early numbers reflect an active promotion push that later stopped, and n=59 cannot support a
trend claim. The motivation for this work is the engagement/conversion gap above, not a decline.

### 1.4 What already exists (do not rebuild)

The zero-result empty state in `frontend/src/components/search/SearchResults.tsx` (~line 386
onward) is already good. It renders, in order:

- `"That searched for the exact phrase. → Match all these words instead"` when the query is quoted
- `"Did you mean X?"` from the API's `suggestion` field
- A full rescue panel with an explanation and one-click suggestion chips

The API already returns `suggestion`, `relaxed_query` and `relaxed_total` on zero-result searches
(`api/internal/handlers/opportunities.go`, the `Search` handler). The frontend already consumes
all three.

**Implication for product strategy:** in-product search repair is already delivered, free and
instantly. Any signup offer framed as "sign up and we'll teach you to search better" trades an
email address for something the user is already being given one second earlier, on the same screen.

### 1.5 Existing signup prompts

`frontend/src/components/SignupPromptModal.tsx` supports three contexts: `bookmark`,
`save-search`, `save-all`. All are **action-triggered** — the user tried to do something requiring
an account. That is the right pattern and should be extended, not replaced.

**Critical gap: nothing is logged when a prompt is shown, dismissed, or clicked.** There is no way
today to tell whether these prompts ever appear, let alone whether they convert. This must be
fixed before any CTA work, or the results will be unmeasurable.

### 1.6 Privacy constraints (from the published policy)

`landing/privacy.html` makes specific commitments that bound the design:

- §8: *"GovTrove does not use cookies for authentication or tracking. Instead, we store
  authentication tokens in your browser's localStorage."* → **a session cookie is out.**
- *"No third-party cookies. None. No tracking you across other websites. No profiling."*
- §1.5: PostHog is *"configured with memory-only persistence"* and *"Events are tied to anonymous
  sessions by default; only after you sign in are events associated with your user profile."*
- §6: on account deletion, *"account data, saved searches, and usage data are deleted... typically
  within 24 hours and no later than 30 days."*

**PostHog cannot deliver the session linkage.** `frontend/src/main.tsx:17` sets
`persistence: 'memory'`, so the PostHog session ID lives only in JS memory and is destroyed on
every page reload — and certainly across the WorkOS redirect. It cannot answer
"two searches → signup → two more searches."

---

## 2. Two bugs found during research

### 2.1 BLOCKER — GDPR account deletion is broken

```
search_events_user_id_fkey  FOREIGN KEY (user_id) REFERENCES users(id)
confdeltype = 'a'   -- NO ACTION
```

`scripts/gdpr.sh:151` runs `DELETE FROM users WHERE id = $USER_ID`. With `NO ACTION` and no
cascade, that statement **fails with a foreign-key violation for any user who has ever searched.**

Currently **19 of 59 users** have `search_events` rows (456 events). Account deletion is therefore
broken for roughly a third of the user base, against a published 24-hour deletion promise.

Fix — use `SET NULL`, not `CASCADE`, so the event survives in de-identified form and aggregate
analytics are preserved:

```sql
ALTER TABLE search_events DROP CONSTRAINT search_events_user_id_fkey;
ALTER TABLE search_events ADD CONSTRAINT search_events_user_id_fkey
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
```

`scripts/gdpr.sh` must also be updated to export and delete the new session data added in §4.

**Ship this independently and immediately. It does not depend on anything else in this document.**

### 2.2 "Did you mean" produces garbage suggestions

`SuggestQuery` (`api/internal/repository/opportunities.go:536`) uses:

```sql
WHERE active = true AND is_latest = true AND word_similarity($1, title) > 0.4
```

The 0.4 threshold is too loose and admits concatenated multi-token junk. Verified live:

| Query | Result | Suggestion returned |
|---|---|---|
| `"Utilization Management"` | 0 | **`visualization managementwayfinding`** |
| `"Beneficiary Travel"` | 0 | `r699--beneficiary Travel` |

The frontend renders this at `text-xl` — large type, high prominence — at exactly the moment the
product is asking the user to trust it. Raise the threshold and reject suggestions that are not
clean token sequences (no missing inter-word spaces, sane length ratio vs. the input).

Fix this **before** placing a signup CTA next to it.

### 2.3 Related observation — quoted-phrase strictness

Not a bug per se, but a large product lever. Quoted queries AND `phraseto_tsquery` with an ILIKE
re-check, so a quoted phrase can return zero while the same words unquoted return thousands:

| Query | Results |
|---|---|
| `"Utilization Management"` | **0** (`relaxed_total: 3440`) |
| `Utilization Management` | 3,440 |
| `"Beneficiary Travel"` | **0** (`relaxed_total: 6`) |
| `Beneficiary Travel` | 6 |

The API already computes `relaxed_total`. Auto-relaxing when `relaxed_total` greatly exceeds zero —
with a visible *"showing results without the exact phrase — [use exact instead]"* notice — would
recover a meaningful share of the 14.8% zero-result rate. This may lift retention more than any
CTA. Consider it after the items above.

A related edge case: a quoted phrase that begins mid-token (e.g. inside an email address) fails the
FTS half while passing the ILIKE half, returning zero. Low user impact; noted for completeness.

---

## 3. Minor API contract inconsistency

The search endpoint returns `"opportunities": null` on zero results, while
`api/internal/handlers/saved_opportunities.go:100` explicitly normalises to `[]`:

```go
if details == nil {
    details = []models.SavedOpportunityDetail{}
}
```

Every current frontend consumer guards with `|| []`, so this is **not a live bug** — but it is an
inconsistent contract and it will bite the next consumer. Normalise the search handler the same way.

---

## 4. Feature A — Session tracking

**Goal:** attribute searches to a session; link a session to a user when they sign up, so a
visitor's pre-signup and post-signup searches can be listed together.

### 4.1 Mechanism

Use **`sessionStorage`**, not a cookie and not `localStorage`.

- A cookie would contradict the published policy (§1.6).
- `localStorage` would create a durable pseudonymous identifier that survives across visits —
  closer to the profiling the policy explicitly disclaims.
- `sessionStorage` is a genuine session: cleared when the tab closes, survives page reloads and
  the WorkOS redirect within the same tab.

There is an exact precedent already in the codebase. `frontend/src/services/api.ts:7`:

```ts
function utmHeaders(): Record<string, string> {
  const campaign = sessionStorage.getItem('govtrove_utm_campaign');
  return campaign ? { 'X-UTM-Campaign': campaign } : {};
}
```

…consumed server-side at `api/internal/handlers/eventlog.go:40`. Mirror this exactly.

`EventLogger.Log(r, userID, event)` is a single choke point that every event passes through —
one place to read the header, one place to set the field.

**Accepted trade-off:** a returning visitor tomorrow is a new session. That is the
privacy-preserving choice.

**Verify during implementation:** that WorkOS AuthKit redirects in the *same tab*. `sessionStorage`
is per-tab; if AuthKit ever uses a popup or a new tab, the signup linkage silently breaks. Test the
exact flow: search → search → sign up → search → search.

### 4.2 Schema — migration `000075`

Use the 6-digit naming convention (highest existing is `000074`; the 3-digit files are legacy).

```sql
ALTER TABLE search_events ADD COLUMN session_id uuid;
CREATE INDEX idx_search_events_session ON search_events (session_id, created_at DESC);

CREATE TABLE session_identities (
    session_id  uuid PRIMARY KEY,
    user_id     integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    linked_at   timestamptz NOT NULL DEFAULT now()
);
```

**Why a link table rather than backfilling `user_id`:** the simpler alternative
(`UPDATE search_events SET user_id = X WHERE session_id = S` on signup) retroactively rewrites
events recorded while the person was anonymous. The link table keeps events as-recorded, makes the
join explicit, and means deleting a single row re-anonymises the pre-signup activity while
preserving aggregate counts.

Include the §2.1 FK fix in this same migration.

### 4.3 The target query

```sql
SELECT e.created_at, e.query, e.total_results,
       e.user_id IS NOT NULL AS was_authenticated
FROM search_events e
JOIN session_identities si ON si.session_id = e.session_id
WHERE si.user_id = $1
ORDER BY e.created_at;
```

Per-session summary for the admin panel:

```sql
SELECT session_id,
       count(*) AS searches,
       min(created_at) AS started,
       max(created_at) AS ended,
       array_agg(query ORDER BY created_at) AS queries
FROM search_events
WHERE event_type = 'search' AND session_id IS NOT NULL
GROUP BY session_id
ORDER BY started DESC;
```

### 4.4 Changes by file

| Layer | File | Change |
|---|---|---|
| Frontend | `services/api.ts` | `sessionHeaders()` beside `utmHeaders()`; mint `crypto.randomUUID()` on first use |
| API | `cmd/api/main.go` (~line 281) | add `X-Session-Id` to CORS `AllowedHeaders` — **easy to miss; the header is silently dropped without it** |
| API | `handlers/eventlog.go` | read header → `event.SessionID`; validate as UUID, reject anything else |
| API | `models/event.go` | add `SessionID *string` |
| API | `repository/events.go` | add to INSERT |
| API | auth path | on first authenticated request carrying a session ID, upsert `session_identities` |
| Admin | `repository/pipeline.go:287` | add `session_id` to the SELECT and to `SearchEventRow` |
| Admin | `handlers/admin.go:279` | accept a `session_id` filter param |
| Admin | `services/api.ts:657` | add `session_id` to `AdminSearchEvent` |
| Admin | `pages/AdminPage.tsx` | render an 8-char prefix; click-to-filter by session |
| Ops | `scripts/gdpr.sh` | include session data in export and delete paths |

### 4.5 Privacy policy updates required

- **§8 Cookies and Local Storage** — disclose the first-party `sessionStorage` analytics session
  identifier, cleared when the tab closes.
- **§1.4 Technical Data** — add the session identifier to the list.
- **Retention** — recommend nulling `session_id` after ~90 days. Sessions stop being analytically
  useful quickly, and this caps how long anything remains linkable.

Nothing here conflicts with "no third-party cookies", "no tracking across websites", or
"no profiling" — this is first-party, same-origin and session-scoped.

---

## 5. Feature B — Signup conversion

### 5.1 Measurement prerequisite (do this first)

Add three event types through the same `EventLogger.Log` choke point, each carrying a `context`
field: `prompt_shown`, `prompt_dismissed`, `prompt_signup_clicked`.

Without these, neither the existing prompts nor any new one can be evaluated. Combined with §4,
this makes session → signup conversion computable for the first time.

**Measure a baseline for at least one week before shipping new CTAs.**

### 5.2 Trigger C — opportunity view (highest expected value, ship first)

1,063 anonymous opportunity views are the strongest unused intent signal in the data. On the
**second** opportunity view within a session:

> Tracking this one? Save it and get alerts on changes to the deadline or documents.
> `[ Save opportunity ]`

Reuses the existing `SignupPromptModal` `bookmark` context. Smallest change, best signal.

### 5.3 Trigger A — zero results

Placed in the existing empty state in `SearchResults.tsx`, **below** the rescue suggestions.
Fix the search first, sell second; if the rescue panel repaired the query, the user never sees this.

> **No matches today.**
> About 2,000 new opportunities are added daily. Get an email the moment something
> matches "*{query}*".

**Corrected 2026-08-25.** An earlier draft said *82 new opportunities are added daily*,
wrong by roughly 25x, and would have shipped a false claim in user-facing copy. Measured
distinct new `notice_id`s per day for the week to 2026-08-25: 1,074 / 2,333 / 2,309 /
2,459 / 2,189 / 1,105 / 2,685. Consistent with CLAUDE.md's stated 500-2,000/day.
Re-measure before committing to a specific number.
> `[ Create a free alert ]` · Takes 20 seconds. No card.

**Why this offer rather than "sign up for search tips":**

- It is genuinely unavailable without an account — a real reason to sign up.
- It is immediate and concrete, not a deferred promise.
- It is honest: zero results usually means nothing matches *today*, and roughly 2,000 new
  opportunities were ingested per day last week.
- The infrastructure already exists (`saved_searches` + `generate-alerts`).
- It reframes failure as timing rather than user error — important, because §2.3 shows the
  product is often the cause.
- It promotes saved searches, which have been used twice ever.

### 5.4 Trigger B — few results (1–5)

Inline below the results, not a modal:

> Only {n} matches. Get notified as new ones are posted. `[ Create alert ]`

### 5.5 Frequency rules

Once per session, dismissible, never shown again after a dismissal. Store the dismissal flag in
the same `sessionStorage` session from §4.

---

## 6. Feature C — LLM search coaching (nightly cron)

The original idea — evaluate users' searches and reach out with tips — is viable as an automated
job. At **~8 candidate searches per day**, cost and scale are non-issues; the binding constraint is
advice quality, not throughput.

### 6.1 Integration

The project already has an LLM tier, via **OpenRouter**, not the Anthropic SDK directly:

- `api/internal/searchrescue/llm.go:20` → `defaultBaseURL = "https://openrouter.ai/api/v1"`
- `api/internal/config/config.go:35` → `SEARCH_RESCUE_MODEL` default `anthropic/claude-haiku-4.5`
- Constructed at `api/cmd/api/main.go:222` via `searchrescue.NewLLMClient(...)`

Reuse that client rather than introducing a second LLM integration path.

### 6.2 Model choice

| Model (OpenRouter id) | Input $/1M | Output $/1M | Use |
|---|---|---|---|
| `anthropic/claude-haiku-4.5` | $1.00 | $5.00 | Existing inline rescue — latency-sensitive |
| `anthropic/claude-sonnet-5` | $3.00 | $15.00 | **Recommended for the nightly digest** |
| `anthropic/claude-opus-5` | $5.00 | $25.00 | If advice quality proves insufficient |

Recommendation: keep Haiku 4.5 for the latency-sensitive inline rescue; use **Sonnet 5** for the
nightly coaching digest, where quality matters and latency does not. At ~8 searches/day the
monthly cost is in the cents either way — do not optimise for it.

Note: because calls route through OpenRouter, the Anthropic Batch API's 50% discount is not
available. At this volume that is irrelevant; do not add a second integration for it.

### 6.3 Design

Run daily, after the ingestion pipeline. For each session from the previous day that contains at
least one zero-result or few-result search **and** belongs to a signed-up user:

1. Gather the session's searches (query text, filters, result counts) via §4's session join.
2. Ask the model to identify the likely intent and produce concrete, specific improvements —
   grounded in real GovTrove capabilities (NAICS/PSC filters, set-aside codes, boolean syntax,
   quoted-phrase behaviour). Prohibit generic advice.
3. Have it produce a rewritten query, and **verify that rewrite against the live search before
   sending** — never email a suggestion that returns zero. This validation step is what separates
   useful coaching from plausible-sounding noise.
4. Send only when the rewrite measurably improves on the original. Silence beats filler.

Rate-limit to at most one coaching email per user per week.

### 6.4 A second, compounding use

The same analysis, aggregated across users, tells you where the product itself is failing —
which is how §2.3-class issues get found. Recurring themes are also good raw material for a public
"search tips" page, which earns SEO and credibility without a manual promise attached.

---

## 7. Recommended sequence

| # | Work | Depends on | Ship separately? |
|---|---|---|---|
| 1 | Purge smoke-test data (§1.1) | — | Yes, immediately |
| 2 | **FK fix + `gdpr.sh` (§2.1)** | — | **Yes, immediately** |
| 3 | Fix `SuggestQuery` garbage (§2.2) | — | Yes |
| 4 | Normalise `opportunities: null` (§3) | — | Yes, trivial |
| 5 | Session tracking (§4) | 2 | Yes |
| 6 | Prompt events (§5.1) | 5 | With 5 |
| 7 | *Measure baseline — one week* | 5, 6 | — |
| 8 | Trigger C, opportunity view (§5.2) | 7 | Yes |
| 9 | Trigger A, zero results (§5.3) | 7, 3 | Yes |
| 10 | LLM coaching cron (§6) | 5 | Yes |
| 11 | Auto-relax quoted phrases (§2.3) | 7 | Evaluate after 7 |

Items 1–4 are independent of everything else and worth doing today. Item 2 is a live compliance
defect.

---

## 8. Open decisions

1. **`sessionStorage` vs `localStorage`** — this document recommends `sessionStorage`. Choosing
   `localStorage` would give cross-visit attribution but creates a durable anonymous identifier
   and needs a more substantial privacy-policy change.
2. **Coaching emails: opt-in or default-on?** Given the product's privacy positioning, an explicit
   opt-in at signup ("send me tips on my searches") is the safer default, at the cost of reach.
3. **Retention window for `session_id`** — 90 days is proposed; not yet decided.
4. **Auto-relaxing quoted phrases (§2.3)** — a behaviour change to search itself. Worth an
   explicit decision rather than folding it into CTA work.

---

## 9. Verification queries used

All research above is reproducible with these. Read-only.

```sql
-- Search outcome distribution (excluding smoke-test contamination)
SELECT count(*) AS searches,
       count(*) FILTER (WHERE total_results = 0) AS zero,
       count(*) FILTER (WHERE total_results BETWEEN 1 AND 5) AS few
FROM search_events
WHERE event_type='search' AND created_at::date < '2026-08-23';

-- Event type breakdown, anonymous vs authenticated
SELECT event_type, count(*) AS n,
       count(*) FILTER (WHERE user_id IS NULL) AS anon
FROM search_events GROUP BY event_type ORDER BY n DESC;

-- The GDPR blocker
SELECT conname, confdeltype   -- 'a' = NO ACTION, 'c' = CASCADE, 'n' = SET NULL
FROM pg_constraint
WHERE conrelid='search_events'::regclass AND contype='f';

SELECT count(DISTINCT user_id) AS users_blocked_from_deletion
FROM search_events WHERE user_id IS NOT NULL;

-- Daily volume for the coaching cron
SELECT round(count(*) FILTER (WHERE total_results = 0)::numeric
             / GREATEST(count(DISTINCT created_at::date),1), 1) AS avg_zero_per_day
FROM search_events
WHERE event_type='search' AND created_at::date < '2026-08-23';

-- Most frequent zero-result queries (coaching input)
SELECT query, count(*) AS times
FROM search_events
WHERE event_type='search' AND total_results = 0
  AND created_at::date < '2026-08-23' AND query <> ''
GROUP BY query ORDER BY times DESC LIMIT 30;
```


---

## 10. What has shipped (2026-08-25)

Audited against production before implementing. Three claims in this document were wrong and
are corrected in place above: the "82 new opportunities daily" CTA figure, the "`save_search`
used twice ever" comparison, and the completeness of the §1.1 purge.

| §7 item | Status |
|---|---|
| 1. Purge smoke-test data | **Done differently** — filtered from all admin displays rather than deleted; rows remain in the table by choice |
| 2. FK fix + `gdpr.sh` | **Done, and larger than described.** §2.1 named one blocking FK; there were four — `search_events`, `mcp_usage`, `invite_link_redemptions`, `gift_code_redemptions`. Migrations `000076`/`000077`, all `SET NULL`. `SET NULL` rather than `CASCADE` on the redemption tables because `max_redemptions` is enforced by `COUNT(*)` with no counter column, so deleting a row would hand back a redemption slot |
| 3. Fix `SuggestQuery` garbage | **Done.** Root cause was not the 0.4 threshold: the query split titles on `\s+` then *deleted* punctuation, gluing `Management/Wayfinding` into `managementwayfinding`. Now splits on punctuation boundaries. The `NN--` PSC prefix cases were left alone — diagnosed 2026-08-17, deliberately not fixed |
| 4. Normalise `opportunities: null` | **Done** — both sites, `repository/opportunities.go` and `repository/saved_searches.go` |
| 5-11 | Not started |

### Prerequisite discovered during implementation

`migrate` could not run at all: `022_geo_synonyms` (legacy 3-digit, added March) collided with
`000022_drop_api_probe_tables`, and golang-migrate refuses to parse a directory containing a
duplicate version. Renumbered to `000075` and made idempotent, since its table already existed
in production from a manual apply. **Anything here needing a migration depended on this first.**

### Privacy

§6's coaching now runs from a local Claude Code session against Anthropic directly, not
OpenRouter. Anthropic is in the privacy policy's processor table, qualified as a consumer
subscription rather than a commercial DPA, with training opt-out noted. Per §13 of that policy
the change **takes effect 2026-09-24** after 30 days' notice — coaching must not run against
other users' data before then. The policy states no name or email is sent to Anthropic, so the
implementation must join on `user_id` and resolve to an email only at send time. Two further
undisclosed processors were found and added: **Resend** (sends every user email; the table had
credited AWS SES) and **PostHog**.
