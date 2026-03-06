# CSV vs API Data Source Analysis (2026-03-06)

## Overview
The active CSV bulk download contains ~28K more records than the API returns (~71K CSV vs ~43K API). All API records are also in the CSV (0 API-only). The CSV-only set is **stable** — not a timing/propagation delay. Over 4 days (Mar 2-6), zero CSV-only records migrated to the API.

## What makes a record CSV-only?

### Type is the biggest differentiator
- **Modification/Amendment/Cancel**: 100% CSV-only (648 records, API never returns this type)
- **Justification and Approval (J&A)**: 100% CSV-only (81 records)
- **Solicitation**: 66% CSV-only (13K of 20K) — the API favors Combined Synopsis/Solicitation over standalone Solicitations
- **Award Notice**: only 2% CSV-only — the API returns almost all award notices
- **Combined Synopsis/Solicitation**: 29% CSV-only

### Age and deadline
- CSV-only median days past deadline: **-133 days** vs **-14 days** for both
- CSV-only skews older but 31% are from 2026, 39% from 2025
- Many have passed deadlines but far-future custom archive dates (`autocustom` archive type)

### Solicitation number relationships
- 79% of CSV-only solicitation numbers also have a sibling record in the API
- **Solicitations**: 99% have an API sibling (the API returns the Combined Synopsis/Solicitation or original notice, CSV also has the standalone Solicitation)
- **Modifications**: 0% have API siblings (their solicitation numbers are completely absent from the API)
- Pattern: API returns one "main" record per solicitation; CSV includes all related records (amendments, modifications, presolicitations)

### `autocustom` archive type
- 67% of all CSV records use `autocustom` (contracting officer sets a custom archive date)
- ~24K autocustom records have archive dates 6 months to 9+ years after the response deadline
- This is why so many active CSV records have passed deadlines — they're intentionally kept active

## API endpoint behavior for CSV-only records
- Search endpoint (`/v2/search?noticeid=...`): returns `totalRecords: 0`
- Individual get endpoint (`/v2/{notice_id}`): returns `404`
- These records genuinely don't exist in the API — not just filtered from search
- **Attachments are unavailable** for CSV-only records (`resource_links` is null)
- Only way to access attachments: SAM.gov `ui_link` (web UI)

## Daily pipeline statistics (Mar 2-6)
- CSV records growing ~700-1500/day (68K → 71K)
- API records growing ~500-1100/day (41K → 43K)
- ~29-30K reconcile DQ issues (stable)
- 4 DQ issues every day (constant)
- Pipeline runs ~25-30 min total

## Pipeline step completion bug (fixed 2026-03-06)
- `ingest-active` Lambda step was stuck as "running" — created the step record but never wrote "completed"
- Root cause: `CompletePipelineStep` used the Lambda context which was nearly expired after processing a large CSV
- Fix 1: All long-running Lambdas now use `context.Background()` with 5s timeout for completion writes
- Fix 2: Status derivation infers completion if a later step already started

## Open questions for later
- Should we try to enrich CSV-only records by fetching from SAM.gov web UI?
- Are the 648 Modifications useful to users or just noise?
- Should we flag CSV-only records in the UI (e.g., "attachments unavailable")?
- The 28K CSV-only records are ~40% of total — significant data the API misses
