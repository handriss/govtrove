# Search API Integration Test Plan

All tests run against a real Postgres 16 container via `make test-api-search`.

## Implemented

- [x] `TestSearch_KeywordSterilizer` — `q=sterilizer` returns 3 results (TEST-001, TEST-003 via description stemming, TEST-006)
- [x] `TestSearch_FilterByType` — `type=o` returns 3 Solicitation results (TEST-001, TEST-005, TEST-008)
- [x] `TestSearch_NoResults` — `q=xyznonexistent123` returns 0 results with correct structure

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
