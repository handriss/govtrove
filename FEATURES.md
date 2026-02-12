# GovTrove — Next Epic Backlog

## 1. Search UX Improvements

Already implemented: notice type filter, set-aside filter, NAICS filter, boolean search (AND/OR), exact-match via quoted strings, exclusions via `-term`.

Still to build:

- [ ] **Searchable agency lookup** — type "Army Corps" or "NAVFAC" or "Fort Liberty" and get all matching opportunities. Pre-map the full hierarchy with aliases, abbreviations, installation names. Uses the free Federal Hierarchy API
- [ ] **NAICS code grouping** — let users define a "NAICS profile" of related codes and search across all with one click. Suggest related NAICS codes based on primary code
- [ ] **Smart notice type defaults** — pre-select the notice types contractors care about (Solicitations, Combined Synopsis, Sources Sought) rather than showing everything. Pure UX improvement
- [ ] **Separate award notices from active opportunities** — SAM.gov mixes all 9 notice types together. Award notices are competitive intelligence (who won, for how much), not actionable bidding opportunities. Show them in a distinct "Awards / Intel" view
- [ ] **"What's new today?" one-click filter** — sort by Last Updated Date, filter to last 1-2 days. Every practitioner guide recommends this daily workflow — make it a single button
- [ ] **PSC (Product Service Code) grouping** — same treatment as NAICS: let users define related PSC profiles, suggest adjacent codes. Public taxonomy data from GSA
- [ ] **"Match my certifications" set-aside toggle** — once company profiles exist, add a one-click toggle: "Only show opportunities I qualify for" based on set-aside eligibility (8(a), SDVOSB, HUBZone, WOSB, etc.). Until profiles exist, offer prominent set-aside quick-filters (partially done)

---

## 2. Save Opportunities

Saved searches already exist (localStorage, Advanced Search page). Individual opportunity bookmarking does not exist yet.

- [ ] Save/bookmark individual opportunities (requires auth)
- [ ] Free tier: 10 saved, paid: unlimited
- [ ] Notes field (editable inline)
- [ ] Filter: Active/Archived
- [ ] Export to CSV (paid feature)
- [ ] Server-side persistence (replace current localStorage saved searches)

---

## 3. Stripe Integration

- [ ] Checkout flow ($9/month or $79/year)
- [ ] Webhook handling (subscription events)
- [ ] Subscription management (upgrade, downgrade, cancel)
- [ ] Signup agreement text ("By creating an account, you agree to...")
- [ ] Free vs. Pro feature gating
