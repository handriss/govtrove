# Search Coaching — Implementation Plan

**Status:** Not started. Blocked on step 0.
**Prepared:** 2026-08-25.
**Supersedes:** §6 of `search-analytics-and-conversion-plan.md`, which assumed an
OpenRouter-backed nightly cron. This runs from a local Claude Code session against
Anthropic directly instead.

**Goal:** find signed-in users whose searches went badly, work out *why*, and prepare a
short email that teaches them the search that would have worked — verified to return
results before it is offered.

---

## 0. GATE — do not start until this is answered

> ### ❓ Has the 30-day privacy notice email been sent, and on what date?
>
> **Ask the user this before doing anything else in this document.** Do not query user
> data, do not draft, do not prototype against real searches.

`landing/privacy.html` was updated on 2026-08-25 to disclose Anthropic as a processor.
Section 13 of that policy commits to:

> *"We will notify you of material changes by sending an email to the address associated
> with your account at least 30 days before the changes take effect."*

The published page states the Anthropic addition **takes effect 2026-09-24**, which assumes
notice went out on 2026-08-25.

| Answer | What to do |
|---|---|
| **Not sent yet** | Stop. The effective date on the live page is wrong and must be corrected to *notice date + 30 days* before anything else. Nothing in this plan may run against other users' data. |
| **Sent on 2026-08-25** | The published date is correct. Everything below is unblocked from **2026-09-24**. Until then, dry-run against the founder's own account only. |
| **Sent on another date** | Update the effective date on the live page to *that date + 30 days*, redeploy the landing site, and treat the new date as the gate. |

Sending the notice is the founder's job — this session never sends it.

---

## 1. Hard constraints

These are published commitments, not preferences. Breaking one is a compliance incident.

1. **No name or email address may be sent to Anthropic.** The processor table says the data
   shared is *"Search query text, filters applied, result counts, and a pseudonymous internal
   user identifier."* Join on `user_id`; resolve to an email address only at draft time,
   locally, after the analysis is finished.
2. **Nothing runs against other users' data before the effective date** (see step 0).
3. **This session never sends email.** It ends at a Zoho draft in `andrew@govtrove.com`, same
   as `user-rescue`. The founder reads and presses send.
4. **Respect `email_preferences`.** Skip any user with `unsubscribed_at IS NOT NULL`. Decide
   which consent flag governs coaching — see §6, open decision 2.
5. **Never offer a search that returns nothing.** Every suggested query is verified against
   the live API first. A coaching email that fails at the link is worse than no email.
6. **At most one coaching email per user per 30 days.** No state table exists for this yet —
   see §6, open decision 3.

---

## 2. Reality check — this population is very small

Measured 2026-08-25, with smoke-test and E2E traffic excluded:

| Population | Poor searches (0 or 1–5 results) | Distinct users |
|---|---|---|
| Signed-in, last 30 days | **31** | **7** |
| Signed-in, all time | 63 | 11 |
| Anonymous, all time | 1,084 | — (unreachable) |

**94% of poor searches are anonymous**, so no email can reach them. The original §6 sized this
work off ~8 poor searches/day, but that figure counted anonymous traffic. The emailable reality
is **roughly one poor search per day, across about 7 people a month.**

Three consequences:

- Cost and throughput are non-issues. Quality is the only thing that matters.
- **This is hand-craftable.** At 7 users a month, a manual pass is viable; the value of this
  feature is the diagnosis quality, not the automation.
- The aggregate use in §5 — spotting where the *product* fails — may be worth more than the
  emails themselves, and it works on the anonymous 1,084 too.

**Decide whether this is worth building at all before building it.** The project is on the
backburner with no paying users; 7 emails a month is a thin return for a new pipeline.

---

## 3. Overlap with `user-rescue` — extend, don't duplicate

`.claude/skills/user-rescue/` already implements almost this exact shape:

> finds users who had a poor experience → drafts a personalized founder-voice email with
> search links **verified to return results** → reviews each draft over Telegram → creates a
> Zoho draft → never sends.

| | `user-rescue` | Search coaching |
|---|---|---|
| Who | Signed up in the **last 24h** | **Any** signed-in user with poor searches |
| Trigger | Signup + bad first visit | Ongoing pattern of failed searches |
| Message | "Here are searches that work for you" | "Here's the technique that would have worked" |
| Ending | Zoho draft, founder sends | Same |

The infrastructure — DB access, API verification helpers, Telegram review, Zoho draft creation,
founder voice — is **already built and proven**. The only genuinely new part is the diagnosis
and the teaching content.

**Strong recommendation:** implement this as a second mode of `user-rescue`, or a sibling skill
that reuses its `scripts/` helpers. Do not stand up a parallel pipeline. Read that skill
end to end before writing anything.

Note the two can collide: a brand-new user with a failed search qualifies for **both**. Whichever
runs first must claim the user so they don't get two founder emails in a week.

---

## 4. Data available — and what is missing

### Session tracking does not exist

§6 of the original plan assumed §4's `session_id` column, so it could group "the searches in one
sitting." **§4 was never implemented.** `search_events` has no `session_id`.

Fallback grouping, good enough at this volume: group a user's searches by `user_id` plus a time
gap — consecutive searches less than ~30 minutes apart are one sitting.

```sql
SELECT user_id, query, filters::text, total_results, created_at,
       created_at - LAG(created_at) OVER (PARTITION BY user_id ORDER BY created_at) AS gap
FROM search_events
WHERE event_type = 'search'
  AND user_id IS NOT NULL
  AND created_at > now() - interval '30 days'
  AND COALESCE(user_agent, '') NOT LIKE 'govtrove-smoke-test%'
  AND COALESCE(user_agent, '') NOT LIKE '%GovTroveE2E%'
ORDER BY user_id, created_at;
```

**The two `user_agent` clauses are mandatory in every query in this plan.** The Go code filters
test traffic from the admin displays (`excludeTestTraffic` in
`api/internal/repository/analytics.go`), but the rows are still in the table and raw SQL will
pick them up. Forgetting this is how the original plan ended up with inflated numbers.

### What is already known about *why* searches fail

Do not re-derive these — they are established and, in two cases, deliberately settled:

- **Quoted phrases are the biggest single cause.** `"Utilization Management"` returns 0 while
  the same words unquoted return ~3,491. The API already returns `relaxed_query` and
  `relaxed_total` on every zero-result search — read them instead of guessing.
- **29% of six-digit NAICS codes have no open opportunities at all.** A user filtering on an
  empty code is not searching wrong; the code is empty. Coaching must say so rather than
  suggest a technique fix. (Real case: a coffee wholesaler ran NAICS 424490 twelve times over
  two weeks, always 0 — federal coffee is bought under 722310/333241.)
- **Sub-6-digit NAICS widens to a prefix; PSC stays exact.** Expected behaviour, not a bug.
- **`NN--` PSC prefixes leak into suggestions** (`r699--beneficiary Travel`). Diagnosed
  2026-08-17, **deliberately not fixed** — don't propose fixing it and don't apologise for it.

### In-product repair already exists

The zero-result empty state already renders a "did you mean", a quoted-phrase escape hatch, and
a full rescue panel with one-click alternatives. **A coaching email must not simply repeat what
the user was already shown one second after their failed search.** It earns its place only by
teaching something the empty state cannot: a pattern across several searches, or the fact that
their chosen code is structurally empty.

---

## 5. Design

Run manually from a Claude Code session in this repo. No cron — all six Codeman crons are
disabled, and at ~1 candidate/day a schedule is not the bottleneck.

**Pipeline:**

1. **Select candidates.** Signed-in users with ≥2 poor searches in one sitting, or the same
   failing query repeated across sittings, in the last 30 days. Skip unsubscribed users, skip
   anyone emailed in the last 30 days, skip anyone `user-rescue` has claimed this week.
2. **Diagnose per user.** Group into sittings; classify each failure — quoted phrase, empty
   NAICS/PSC code, too many terms, typo, over-tight filter, or *nothing matches today*.
   Use `relaxed_total` and the code-population data rather than guessing.
3. **Only send `user_id`-keyed data to Anthropic** — query text, filters, result counts. No
   email, no name (constraint 1).
4. **Produce a rewritten query, then verify it.** Hit the live API. If the rewrite returns 0,
   it is not a suggestion — discard it and try again.
5. **Drop anyone with nothing genuinely useful to say.** Silence beats filler. Expect to drop
   most candidates; that is a success condition, not a failure.
6. **Draft in the founder's voice**, one concrete improvement, links that work.
7. **Telegram review**, per draft, same as `user-rescue`.
8. **Zoho draft only** after explicit approval. Never send.

**Model:** whatever the Claude Code session runs on. No model configuration, no
`SEARCH_RESCUE_MODEL`, no `OPENROUTER_API_KEY` — those belong to the separate inline
search-rescue feature and are untouched by this.

---

## 6. Open decisions

1. **Is this worth building?** 7 users/month, against an existing empty state that already
   repairs searches in-product. The aggregate analysis (§7) may be the better half.
2. **Which consent flag governs coaching?** `email_preferences` has `search_alerts` and
   `opportunity_alerts`; neither means "send me search tips." Options: reuse `search_alerts`
   (arguably a stretch), add a column, or treat coaching as transactional founder correspondence
   rather than a marketing send. **Leaning: add an explicit flag** — the privacy positioning
   makes silent reuse of an unrelated consent flag a bad look.
3. **Where is "already emailed" state kept?** `sent_emails` exists and is the obvious place if
   coaching sends carry a distinguishable type. Confirm before relying on it.
4. **Anonymous users are 94% of the problem and cannot be emailed.** The only route to them is
   in-product (the existing empty state) or §4/§5 of the original plan. Out of scope here — but
   worth remembering that this feature addresses the small half.

---

## 7. The compounding use — probably the better half

The same diagnosis, aggregated across *all* failed searches including the 1,084 anonymous ones,
says where the **product** is failing rather than the user. That has no privacy gate on the
aggregate (no personal data leaves), no consent question, no email to draft, and a much larger
sample.

Recurring themes are also raw material for a public "search tips" page, which earns SEO without
promising anyone anything.

**If only one half of this gets built, build this one.**

---

## 8. Sequence

| # | Step | Gate |
|---|---|---|
| 0 | **Ask whether the 30-day notice was sent, and when** | — |
| 1 | Correct the effective date on the live policy if needed | 0 |
| 2 | Decide open decision 1 — build it at all? | 2 (§2 volume) |
| 3 | Read `.claude/skills/user-rescue/` end to end | 2 |
| 4 | Resolve consent + dedupe state (decisions 2, 3) | 3 |
| 5 | Build diagnosis + verification against founder's own account only | 4 |
| 6 | First real run | **effective date** |
| 7 | Aggregate product-failure analysis (§7) | none — can start any time |

Step 7 is not blocked by step 0: it touches no personal data and sends no email.
