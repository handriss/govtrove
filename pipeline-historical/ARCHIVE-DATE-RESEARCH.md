# Archive Date Research (2026-03-01)

Research into the `archive_date` column across `snap_csv` and `snap_api` tables, based on the latest snap_csv batch (run `a5fa6abf`, Feb 28 2026, 69,719 records).

---

## 1. Does `snap_api` have an `archive_date` column?

**No.** The `snap_api` table has no dedicated `archive_date` column. It stores: `id`, `run_id`, `notice_id`, `raw_data` (JSONB), `content_hash`, `snapshot_date`, `created_at`. If the API response includes an archive date, it's embedded in `raw_data` but not broken out.

---

## 2. How is `archive_date` populated in `snap_csv`?

- Column type is **TEXT** (nullable) — stores the raw CSV string value
- `nilIfEmpty()` converts empty strings to SQL `NULL` on insert
- In `snap_csv`, the value is stored as-is from the CSV (e.g. `"2026-03-04"`)

### Parsing rules (in reconcile, when building Opportunity structs)

| Input | Result | DQ issue recorded? |
|-------|--------|--------------------|
| Empty string | `nil` | No |
| Sentinel date (year < 1980, e.g. `12/31/1969`) | `nil` | Yes (`sentinel_date`) |
| Unparseable string | `nil` | Yes (`unparseable_date`) |
| Valid date | `*time.Time` | No |

The parsed value flows into the `opportunities` table as `TIMESTAMPTZ` (nullable).

---

## 3. Latest snap_csv batch overview

**Run:** `a5fa6abf` — Feb 28, 2026 04:45 UTC — 69,719 records — 14 min duration

| Metric | Count |
|--------|-------|
| Total records | 69,719 |
| Has `archive_date` | 67,865 (97.3%) |
| NULL `archive_date` | 1,854 (2.7%) |

Most records have a future archive_date (scheduled archival). Top dates:

| archive_date | count |
|--------------|-------|
| 2026-03-04 | 2,017 |
| NULL | 1,854 |
| 2026-02-28 | 1,627 |
| 2026-03-07 | 1,385 |
| 2026-03-19 | 1,310 |
| 2026-03-14 | 1,219 |
| ... | ... |

---

## 4. Deep dive: records with NULL archive_date

### By opportunity type

| Type | Count |
|------|-------|
| Presolicitation | 679 |
| Special Notice | 608 |
| Combined Synopsis/Solicitation | 156 |
| Solicitation | 134 |
| Sources Sought | 96 |
| J&A | 81 |
| Justification | 80 |
| Award Notice | 18 |
| Modification/Amendment/Cancel | 2 |

### By archive_type

| archive_type | Count |
|--------------|-------|
| `autocustom` | 1,304 |
| `auto15` | 527 |
| NULL | 13 |
| `auto30_j` | 10 |

### Date ranges

| Metric | Value |
|--------|-------|
| Earliest posted_date | 2008-04-15 |
| Latest posted_date | 2025-12-19 |
| Earliest response_deadline | 2008-04-10 |
| Latest response_deadline | 2028-04-13 |

### Key observations

- **These are old, stale opportunities.** Mostly from the 2016-2017 era — zombie records sitting in the active CSV for years.
- **Most (1,841/1,854) have an `archive_type` but no `archive_date`.** SAM.gov knows the archiving rule (e.g. `autocustom`, `auto15`) but hasn't computed an actual date. Likely because the response deadline was never set, so auto-archive couldn't calculate.
- **The 13 with both NULL `archive_type` and NULL `archive_date`** are all **Award Notices** with no response deadline — posted and never configured for archiving. They'll sit in the active feed indefinitely.

### The 13 fully-NULL records (sample)

All Award Notices, examples:
- "IDIQ MACC SMALL BUSINESS SET-ASIDE FOR NEW CONSTRUCTION..." (posted 2025-12-19)
- "Complementary Positioning, Navigation, and Timing (CPNT) Services Rapid Phase II" (posted 2025-08-26)
- "F108 Engine Digital Turning Tool Kits" (posted 2024-09-17)

---

## 5. Solicitation number sharing in snap_csv

### Overall

- **10,162 solicitation numbers** appear more than once
- **45,765 records** (66% of total) share a solicitation number with at least one other record
- Each record has a unique `notice_id` — these are separate SAM.gov entries (amendments, awards, modifications) under the same procurement

### Examples

**`47QSMD20R0001`** (7,089 records) — GSA "Multiple Award Schedule". Each record is a separate Award Notice. Archive dates set 5 years out (2031).

**`N4008025R9500`** (77 records) — Navy MACC solicitation. Same title, same archive_date, but each is a separate modification posted over months.

### Among NULL archive_date records

- **1,854 records** with no archive_date
- **702 distinct solicitation numbers** (plus 206 with no solicitation number at all)
- Many solicitation numbers have clusters of 8-19 notices each
- Mostly Navy (`N6...`, `N4...`) and DoD (`FD2020...`) from 2016-2019

Top shared solicitation numbers (no archive_date):

| solicitation_number | count |
|---------------------|-------|
| N6274219R1199 | 19 |
| N4008517R0309 | 16 |
| N6134016R1146 | 16 |
| FD2020-22-00584 | 13 |
| N6945019R1900 | 13 |

---

## 6. Archive date consistency across solicitations

**Question:** Does archive_date stay the same across all notices in a solicitation?

### Results

Of 9,788 solicitations with multiple notices:
- **3,539 (36.2%)** — all notices share the same archive_date
- **6,249 (63.8%)** — notices have different archive_date values

### Why they vary

Each notice under a solicitation is its own SAM.gov entry (presolicitation → solicitation → amendments → award). Each gets its own archive date based on when *that notice* was posted and what archive rule applies. They're not "the same opportunity changing its archive date" — they're separate notices with independent lifecycles.

### Conclusion

**Each `notice_id` has its own `archive_date`.** Archiving logic should operate at the notice_id/opportunity level, not at the solicitation level. The solicitation number is a grouping/display concern, not a uniqueness concern for active/inactive status.
