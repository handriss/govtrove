# GovTrove — Testing Strategy

## Unit Tests

### Backend — API (`api/`)

| Area | What to test |
|------|-------------|
| `handlers/opportunities.go` | Search param parsing, query validation, pagination math, filter normalization |
| `handlers/contact.go` | Email validation, honeypot detection, input sanitization |
| `handlers/account_request.go` | Request type validation, auth requirement enforcement |
| `handlers/analytics.go` | Period parsing (24h/7d/30d), response shaping |
| `middleware/auth.go` | JWT claim extraction, issuer validation, expired token handling, missing token behavior |
| `models/*` | Struct validation, default values |

### Backend — Ingestion (`ingestion/`)

| Area | What to test |
|------|-------------|
| `samgov/csv.go` | `mapRowToOpportunity` with various CSV shapes, `parseDate` with all 12 formats, `parseAmount` edge cases (`$1,234.56`, empty, negative), `parseActive` variants, `sanitizeToUTF8` with Windows-1252 chars, MIN_POSTED_DATE filtering during parse |
| `samgov/api.go` | API response parsing, pagination logic, opportunity mapping from API format |
| `ingestion/service.go` | Stats accumulation, cross-reference logic (`buildNoticeIDSet`, `crossReference`), `compareOpportunity` mismatch detection, notification message formatting |
| `config/config.go` | `GetMinPostedDate` parsing, mode validation, defaults |

### Frontend (`frontend/`)

| Area | What to test |
|------|-------------|
| `pages/AdvancedSearchPage.tsx` | `parseFiltersFromParams` — URL to filter state mapping, default types logic, `isDefaultTypes` check, query builder group/term manipulation |
| `pages/SimpleSearchPage.tsx` | `buildParams` output, set-aside state from URL, "What's New Today" date calculation |
| `components/QueryBuilder.tsx` | Group add/remove, term add/remove, operator toggling, edge cases (single group, max groups) |
| `components/FilterPanel.tsx` | Filter count calculation, checkbox state, accordion behavior |
| `components/SetAsideChips.tsx` | Toggle on/off, multi-select, initial state from props |
| `hooks/useSearch.ts` | State transitions (idle → loading → results/error), pagination updates, reset behavior |
| `hooks/useDebounce.ts` | Timing behavior |
| `services/api.ts` | Request URL construction, param serialization, error handling, auth header injection |

---

## Integration Tests

### Database + Repository (API side)

| Area | What to test |
|------|-------------|
| `repository/opportunities.go` | Full-text search returns ranked results, filter combinations (type + set-aside + NAICS + state + date range), pagination (page 1, last page, beyond range), empty result handling, `GetFilterOptions` returns distinct values |
| `repository/users.go` | Upsert creates new user, upsert updates existing (idempotent on workos_id), plan field persists |
| `repository/events.go` | Event insertion with all fields, session_id grouping |
| `repository/analytics.go` | Aggregation queries return correct counts across time periods, zero-result term detection |
| Search vector trigger | Insert opportunity → verify `search_vector` tsvector is populated, update title → verify vector updates |

### Database + Ingestion

| Area | What to test |
|------|-------------|
| `database/opportunities.go` | `BatchUpsertOpportunities` — insert new, update existing, mixed batch, duplicate notice_id handling, `RunStats` accuracy |
| `database/samgov_requests.go` | Request logging with all fields, response time tracking |
| `database/csv_cache.go` | Cache headers round-trip (save → retrieve), `UpdateCSVCacheLastChecked` timestamp |
| Ingestion run lifecycle | Create run → complete with stats → verify columns (`records_skipped`, `total_db_count`), create run → fail with error message |

### API Handler + Database (HTTP integration)

| Area | What to test |
|------|-------------|
| `GET /api/opportunities?q=...` | End-to-end: seed DB → HTTP request → verify JSON response shape and content |
| `GET /api/opportunities/{id}` | Valid ID returns 200 with full detail, invalid ID returns 404 |
| `GET /api/filters` | Returns real distinct values from seeded data |
| `POST /api/auth/sync` | Valid JWT → user created in DB, subsequent call → user updated |
| `POST /api/contact` | Valid input → 201 + row in DB, honeypot filled → rejected |
| `POST /api/account/requests` | Requires auth, creates request in DB, publishes to SNS (mock) |
| Rate limiting | 101st request within a minute → 429 |

### SAM.gov Client (with mock server)

The codebase has `cmd/mockserver/main.go` for this purpose.

| Area | What to test |
|------|-------------|
| CSV client | Download → parse → verify opportunity count and field mapping, conditional request with ETag → 304 handling, malformed CSV rows → graceful skip |
| API client | Pagination across multiple pages (verify delay between pages), response mapping, error responses (429, 500) |
| Description client | Batch description fetch, partial failures |

---

## E2E Tests

Browser-based tests (Playwright recommended):

| Flow | Steps |
|------|-------|
| **Search → Detail** | Load Simple Search → type query → verify results appear → click result → verify detail page shows correct data → verify "Back to results" works |
| **Advanced Search filters** | Load Advanced Search → verify default types are pre-selected (4 of 7) → add set-aside filter → verify results narrow → clear all → verify all results return |
| **What's New Today** | Click "What's New Today?" → verify results sorted by posted date desc → verify date filter applied |
| **Set-aside chips** | Click "8(a)" chip → verify chip highlights → verify API call includes `set_aside=8A` → click again → verify deselected |
| **URL persistence** | Search with filters → copy URL → open in new tab → verify same filters and results |
| **Auth flow** | Sign in → verify profile button appears → navigate to Profile → verify user info → sign out → verify signed out state |
| **Contact form** | Fill form → submit → verify success message |
| **Mobile responsive** | Run key flows at 375px viewport — search, filters, detail page |

---

## Contract Tests (API ↔ Frontend)

Define a shared schema for API responses and validate both sides against it. Catches drift between what the backend returns and what the frontend expects. Cover:
- `SearchResult` shape (opportunities array, total, page, totalPages)
- `Opportunity` vs `OpportunityListItem` fields
- `FilterOptions` structure
- Error response format

---

## Smoke Tests (Post-Deploy)

Run after every `make deploy-*`:

| Check | How |
|-------|-----|
| API health | `curl https://api.govtrove.com/health` → 200 |
| Search works | `curl https://api.govtrove.com/api/opportunities?q=test&limit=1` → 200 with results |
| Frontend loads | `curl https://app.govtrove.com` → 200 with `<div id="root">` |
| Landing loads | `curl https://govtrove.com` → 200 |
| OG tags work | `curl https://api.govtrove.com/og/opportunities/{known-id}` → 200 with meta tags |

---

## Data Quality Tests (post-ingestion)

Query the DB after each ingestion run:

| Check | Query |
|-------|-------|
| No null titles | `SELECT count(*) FROM opportunities WHERE title IS NULL` → 0 |
| No empty notice_ids | `SELECT count(*) FROM opportunities WHERE notice_id = ''` → 0 |
| Valid types | `SELECT DISTINCT type FROM opportunities` → only known types |
| Search vector populated | `SELECT count(*) FROM opportunities WHERE search_vector IS NULL` → 0 |
| Reasonable date range | No `posted_date` in the future or before 2020 |
| Storage usage | `SELECT pg_database_size('neondb')` < threshold |

---

## Load Tests (pre-launch)

| Scenario | Tool | Target |
|----------|------|--------|
| Concurrent search | k6 or hey | 50 concurrent users, sustained 1 min → p99 < 500ms, 0 errors |
| Large result sets | k6 | Search with no filters (broad query) → no timeout |
| Rate limit enforcement | k6 | 200 req/min from single IP → first 100 succeed, rest get 429 |

---

## Priority Order

1. **Unit tests for CSV parsing** — highest ROI, this is where data bugs originate
2. **Integration tests for search repository** — the core product feature
3. **Smoke tests** — cheap, catch deployment failures immediately
4. **E2E for search → detail flow** — the primary user journey
5. **Data quality tests** — catch ingestion regressions
6. Everything else

---

## Tooling

| Layer | Tool |
|-------|------|
| Go unit/integration | `go test` + `testify` for assertions |
| Go DB integration | `testcontainers-go` with PostgreSQL, or Neon branching (create test branch per run) |
| Frontend unit | Vitest (already Vite-based) + React Testing Library |
| E2E | Playwright |
| API contract | Zod schemas shared or generated from Go types |
| Load | k6 (scriptable, good for CI) |
| Smoke | Simple shell script or GitHub Actions step |
