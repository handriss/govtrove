# SEO recheck plan — set 2026-08-18

Three waves of SEO work shipped in August 2026. They have different propagation
clocks, so they get measured on different dates. **Do not rewrite titles before the
first recheck** — measuring mid-flight is what this plan exists to prevent.

Origin session (Claude Code, same repo): `f43f57ce-65b1-4871-a423-f57d188e87d6`
→ `claude --resume f43f57ce-65b1-4871-a423-f57d188e87d6` (local transcript; may be
pruned eventually — this file is the durable record).

---

## What shipped

| Date | Commit | Change |
|---|---|---|
| 2026-08-04 | `f004203` | CTR title/meta rewrites on 3 blog posts + count-aware NAICS template |
| 2026-08-08 | `5874322` | 1,758 PSC pages, PSC/NAICS hubs, definition-first titles, noindex under 3 opportunities |
| 2026-08-18 | `564cf21` | Internal linking: sibling "Related codes" blocks + PSC↔NAICS cross-links |

## Recheck dates

- **~2026-09-01** — waves 1 and 2 (4 weeks / 3.5 weeks out)
- **~2026-09-29** — wave 3, internal linking (6 weeks out; crawl-priority effects are slow)

---

## Baselines to compare against

All from Google Search Console, measured 2026-08-18. GSC lags ~3 days.

**90-day totals (2026-05-16 → 2026-08-14):** 272 clicks / 103,360 impressions /
**0.26% CTR** / avg position ~8.

**Section breakdown (90d):**

| Section | Pages | Impressions | Clicks | CTR | Avg pos |
|---|---|---|---|---|---|
| Blog | 13 | 77,667 | 156 | 0.20% | 7.5 |
| /contracts/naics/* | 950 | 11,045 | 29 | 0.26% | 8.3 |
| /contracts/psc/* | 657 | 5,336 | 17 | 0.32% | 7.4 |
| Code finders | 2 | 5,179 | 24 | 0.46% | 16.4 |
| /contracts/agency/* | 85 | 1,335 | 4 | 0.30% | 14.4 |
| Homepage | 1 | 1,165 | 36 | **3.09%** | 16.9 |

**11d before vs 11d after the Aug 4/8 waves** (Jul 25–Aug 4 vs Aug 5–15):
impressions **9,614 → 20,552 (+114%)**, clicks **31 → 69 (+123%)**. Every section up.

**Per-page (the pages that were rewritten):**

- `sam-gov-data-services-csv-api-explained.html` — before 15 clicks/6,696 impr (14d),
  after 23 clicks/8,270 impr (11d). **CTR +24%, clicks/day +95%.** The clean win.
  Recrawled Aug 7.
- `govtribe-vs-govtrove.html` — impressions doubled (746→1,512), position 7.8→6.4,
  but CTR fell 0.40%→0.13% and clicks stayed flat (3→2). Recrawled Aug 6.
- `sam-gov-foreign-entity-registration.html` — **measurement invalid**, only recrawled
  Aug 12 so the "after" window was mostly still the old title. Re-measure properly.

**Internal links (wave 3):** ~5,600 → **39,092** unique internal links across leaf
pages. 1,021/1,030 PSC leaves and 612/612 NAICS leaves are linked from their parent.

---

## What to check on each date

1. Re-run the section breakdown and the before/after comparison (scripts: the
   `weekly-content` GSC helper `.claude/skills/weekly-content/scripts/gsc.sh --days 90`,
   and `.claude/skills/health-check/scripts/gsc_health.py` for the 7d trend).
2. **Verify recrawl before trusting any per-page CTR change** — use the URL Inspection
   API for `lastCrawlTime`. A page recrawled after the change has no valid "after" data.
   This caught two false readings in August.
3. For wave 3 specifically: has the *number of pages earning impressions* grown?
   Baseline was ~657 PSC + 318 NAICS of 1,030 + 612 published. Internal linking should
   move that count, not just CTR.

---

## Known non-issues — do not "fix" these

- **S3 has ~2,972 pages, the sitemap has 1,833.** By design: `indexThreshold = 3` in
  `pipeline/cmd/lambda/seo-pages/main.go`. Thin pages are generated but noindex'd and
  excluded. Verified: in-sitemap pages are indexable, excluded ones return
  `noindex,follow`.
- **No orphan-page problem.** A 2026-08-18 claim of "712 orphans" was a bad regex
  (`psc/[0-9]+\.html` misses letter-prefixed service codes like R425/J019/Y1GA, which
  are most of them). Coverage is effectively complete.
- **`TÜRK?YE` mojibake** is SAM.gov's, already byte `0x3f` in our DB. Windows-1252 has
  `Ü` but no `İ`. Not ours to fix.

## Still open (not blocked by the SEO clock)

- **Code finders at position 16.4** with 5,179 impressions / 90d and the second-best
  CTR on the site. Two pages. Best remaining growth action.
- **SEO pages show closed contracts.** e.g. `/contracts/psc/8405.html` is titled
  "21 Active Contracts" and lists items with 2023 deadlines: the CO set `archive_date`
  to 2028 so the expiry sweep never fires, and the staleness sweep only applies when
  `archive_date IS NULL`. Consider ranking/counting on `response_deadline`.
- **`govtribe-vs-govtrove`** — diagnose query intent before rewriting again. "govtribe
  pricing" searchers may be existing GovTribe customers no title can win.
- Monitoring gaps: no "pipeline hasn't run" alarm (CloudWatch `notBreaching` hides a
  stopped scheduler), no uptime monitor on `govtrove.com` or `mcp.govtrove.com`,
  BetterStack notification routing unverified.
