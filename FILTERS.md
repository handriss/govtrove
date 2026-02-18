# Filter Status & Reintroduction Plan

## Current State (Phase 1)

The opportunities table has ~85K records from the CSV pipeline. The API was originally written for API-sourced column names, creating ~6 column mismatches. Phase 1 gets search + type working; all other filters are disabled.

## Column Mismatches (API → Pipeline DB)

| API Column (broken) | Pipeline Column (actual) | Status |
|---|---|---|
| `full_parent_path_name` | `department` | Fixed in Phase 1 |
| `pop_state_code` | `pop_state` | Fixed in Phase 1 |
| `pop_city_name` | `pop_city` | Fixed in Phase 1 |
| `pop_country_code` | `pop_country` | Fixed in Phase 1 |
| `naics_codes` (array) | Does not exist | Removed in Phase 1 |
| `data_source` | `data_sources` | Fixed in Phase 1 |

## Missing Tables

| Table | Status |
|---|---|
| `opportunity_resources` | Does not exist — `getResourceLinks` stubbed to return nil |

## Filter Inventory

| Filter | API Param | DB Column | Phase 1 Status | Notes |
|---|---|---|---|---|
| Full-text search | `q` | `search_vector` | Working | `websearch_to_tsquery` |
| Type | `type` | `type` | Working | Frontend sends full names ("Solicitation"), DB has full names |
| Posted date | `posted_from/to` | `posted_date` | Working | Used by "What's New Today?" |
| Response deadline | `deadline_from/to` | `response_deadline` | Working | SimpleSearchPage auto-filters to future deadlines |
| Set-aside | `set_aside` | `set_aside_code` | Disabled | Needs column verification + code mapping |
| NAICS code | `naics` | `naics_code` | Disabled | Single column only (no array column) |
| State | `state` | `pop_state` | Disabled | Column name was wrong (`pop_state_code`) |

## GetFilterOptions

Stubbed to return empty `FilterOptions{}`. Will be rebuilt per-filter as each is reintroduced.

The old implementation had a `typeLabels` map converting single-letter codes → full names, but the pipeline writes full names directly. The map was wrong for our data.

## Reintroduction Plan

Each filter will be reintroduced individually so it can be tested and UX-evaluated:

1. **Phase 1** (this PR): Full-text search + Type filter
2. **Phase 2**: Set-aside filter (verify codes match between frontend chips and DB values)
3. **Phase 3**: State filter (fix column name, verify state codes)
4. **Phase 4**: Date filters (posted date, response deadline)
5. **Phase 5**: NAICS code filter
6. **Phase 6**: GetFilterOptions — rebuild with correct column names and live counts
7. **Phase 7**: Resource links (when `opportunity_resources` table exists)
