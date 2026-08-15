# Search API Integration Test Plan

All tests run against a real Postgres 16 container via `make test-api-search`.

## Implemented

- [x] `TestSearch_KeywordSterilizer` — `q=sterilizer` returns 3 results (TEST-001, TEST-003 via description stemming, TEST-006)
- [x] `TestSearch_FilterByType` — `type=o` returns 3 Solicitation results (TEST-001, TEST-005, TEST-008)
- [x] `TestSearch_NoResults` — `q=xyznonexistent123` returns 0 results with correct structure

### Added 2026-08-14 (fixtures TEST-011..015, taken from real zero-result searches)
- [x] `TestSearch_QuotedStopwordOnlyPhrase` — `"IT"` returns results (empty tsquery must not veto the literal re-check)
- [x] `TestSearch_ActiveOnlyKeepsUndatedNotices` — `active=true` keeps notices with no deadline
- [x] `TestSearch_ActiveOnlyStillExcludesExpired` — `active=true` still drops closed notices
- [x] `TestSearch_KnownItemSolicitationLookup` — dashed / undashed / lowercase all find the notice past its deadline
- [x] `TestSearch_KnownItemDoesNotHijackTopicSearch` — a topic search must not match a notice number
- [x] `TestSearch_QuoteCollapseOffersRelaxedQuery` / `TestSearch_NoRelaxedHintWhenUnquoted`

### Added 2026-08-15 (fixture TEST-016)
- [x] `TestFacets_TotalMatchesSearchTotal` — facets and search must agree. **Caveat: this pins API-side parity only.** The 2026-08-14 production bug was the *frontend* serializer omitting `active`; no Go test can catch that.
- [x] `TestFacets_HonoursActiveFilter`
- [x] `TestSearch_ExplicitDeadlineRangeExcludesUndatedNotices` — an explicit range is a different intent from "still open"
- [x] `TestSearch_QuotedStopwordPhraseMatchesSubstrings_CURRENT` — **pins today's substring semantics**: `"IT"` also matches Monitor/Unit/the pronoun. Moving to case-sensitive word-boundary matching MUST break this test; update it deliberately.

Unit-level (in `internal/repository`): `TestBuildOrderClause_*` (relevance default, browsing default, fallback, explicit sort wins), `TestLooksLikeSolicitationNumber` (14 boundary cases), `TestNormalizeSolNum`.
Rescue (in `internal/searchrescue`): `TestNormalizeForProbe_*`, `TestParamsMapRoundTrip_PreservesActiveOnly`.

## Known gaps (2026-08-15)

- **No frontend tests exist at all** — no vitest/jest/testing-library in `package.json`. Two of the three regressions this session were frontend-only: the facets serializer omitting `active`, and sort being pinned by `localStorage`. Neither is reachable from Go. `services/api.ts` has **three** param serializers (search, rescue, facets) that must stay in sync.
- `quoteRelaxThreshold` (no relax probe when results are plentiful) is unpinned — the seed is too small to produce 10+ hits for one phrase.
- Pagination, visibility and most filter permutations below are still unimplemented.

## Planned

### Keyword Search
- [ ] Phrase search (`q="supply chain"`)
- [ ] OR search (`q=sterilizer OR cybersecurity`)
- [ ] Multi-word search (`q=cloud migration`)
- [ ] Empty query (returns all active/latest)
- [ ] Long query (>500 chars truncated)
- [ ] Special characters in query
- [ ] Description match (`q=penetration testing`)
- [ ] Solicitation number match (`q=SOL-2026-002`)
- [ ] Exclusion/negative keyword

### Filters
- [ ] Type multi-value (`type=o,p`)
- [ ] Set-aside filter (`set_aside=SBA`)
- [ ] Set-aside NONE filter
- [ ] NAICS exact (`naics=339113`)
- [ ] NAICS prefix (`naics_prefix=339`)
- [ ] PSC exact (`psc=J065`)
- [ ] PSC prefix (`psc_prefix=J`)
- [ ] Agency/department filter
- [ ] State single (`state=VA`)
- [ ] State multi (`state=VA,DC`)
- [ ] Solicitation number filter (`sol_num=SOL-2026`)
- [ ] Pop city filter
- [ ] Posted date from (`posted_from=2026-03-01`)
- [ ] Posted date to (`posted_to=2026-03-01`)
- [ ] Deadline date from
- [ ] Deadline date to
- [ ] Posted date range
- [ ] Deadline date range
- [ ] NAICS prefixes multi (`naics_prefixes=339,541`)

### Combined Filters + Search
- [ ] Keyword + type (`q=sterilizer&type=o`)
- [ ] Keyword + set_aside + state
- [ ] All filters combined

### Sorting
- [ ] Default sort (posted_date DESC)
- [ ] Relevance sort with query
- [ ] Deadline sort
- [ ] Title sort
- [ ] Department sort
- [ ] Invalid sort field (falls back to default)
- [ ] Order direction (ASC vs DESC)

### Pagination
- [ ] Default limit (25)
- [ ] Custom limit (`limit=2`)
- [ ] Page 2 (`page=2&limit=3`)
- [ ] Beyond results (`page=100`)
- [ ] Limit bounds (>100 capped, 0 ignored)
- [ ] Total pages math
- [ ] Page cap at 10000

### Visibility
- [ ] Inactive excluded (TEST-009 never appears)
- [ ] Old versions excluded (TEST-010 never appears)
- [ ] Unfiltered count is 8 (not 10)

### Suggestions
- [ ] Misspelled query returns suggestion
- [ ] Correct query returns no suggestion
- [ ] Suggestion only on zero results

### Edge Cases
- [ ] Empty request (no params)
- [ ] Empty param values (`q=&type=`)
- [ ] SQL injection attempt (`q='; DROP TABLE--`)
- [ ] Large page number
