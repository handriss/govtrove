# Browser end-to-end checks

> Named `test-ui` because `make test-e2e` already means the pipeline's Ginkgo suite.

Drives the real UI in a browser — typing into the search box, clicking the
buttons — against **production by default**. Every fix from 2026-08-14/15 has a
check here, so the same regressions can't come back silently.

```bash
make test-ui             # both viewports against production
make test-ui-headed      # watch it run
cd e2e && npx playwright show-report   # last HTML report
```

First run only: `cd e2e && npm install && npx playwright install chromium`.

## Pointing it somewhere else

```bash
E2E_BASE_URL=http://localhost:5173 E2E_API_URL=http://localhost:8080/api make test-ui
```

## What each check defends

| Check | Regression it catches |
|---|---|
| quoted all-stopword phrase returns results | `"IT"` returned **0** — an empty tsquery vetoed the ANDed literal re-check |
| headline count agrees with the search results | facets and search are **separate endpoints with separate param serializers**; when facets dropped `active` the page showed "26 results" above a list of 12 |
| first-time visitor gets relevance ranking | sort is persisted in `localStorage`; every prior visitor was pinned to `posted_date` by a value they never chose |
| pasted notice number found regardless of filters or dashes | a notice number must bypass the open-only and type filters, and match with or without dashes |
| quote collapse is explained and the count is honest | quotes can cut thousands of hits to zero; the banner must say so and clicking it must deliver the number it promised |
| zero-result search is rescued with a count that holds up | rescue suggestions carry **server-verified** counts — the number on the chip must be the number you get |
| active-only keeps undated notices | `response_deadline >= today` silently dropped ~8k notices with no deadline, because `NULL >= x` is NULL |

Both projects run every check: `desktop` (1440×900) and `mobile` (390×844,
Chromium emulation). Mobile is a genuinely different component tree — bottom
nav, full-screen filter sheet — not just narrower CSS.

## Design rules, so these stay useful

**Never assert a fixed count.** These run against live data that changes daily.
`"Utilization Management"` returned 1 result on Aug 14 and 0 on Aug 15 when that
notice closed. Assert invariants instead: non-zero, parity between two numbers,
a promised count matching the delivered one.

**Discover fixtures from the API at run time.** The notice-number check queries
for a currently-closed notice rather than hard-coding one that will age out.

**Skips are honest.** A check that can't find a valid fixture skips loudly
rather than passing vacuously.

## These searches land in your analytics

Runs create real `search_events` rows and PostHog events. The user agent carries
a marker so they can be excluded:

```sql
WHERE user_agent NOT LIKE '%GovTroveE2E%'
```

Worth remembering before reading the zero-result rate — the rescue check
deliberately performs a zero-result search every run.

## Known gap

The `services/api.ts` serializers (search, rescue, facets) are still kept in
sync by hand. The count-parity check catches drift for `active`; a new filter
added to only one serializer would need its own check.
