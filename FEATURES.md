# GovTrove — Next Epic Backlog

## 2. Search UX Improvements

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

## 3. Opportunity Detail Page

Most metadata is already displayed: solicitation number, notice type, dates (posted/response/archive), set-aside with color-coded badges, NAICS codes, agency/department hierarchy, place of performance, award info (number, amount, date, awardee), attachments/resources with links, "View on SAM.gov" link, expandable description.

Still to build:

- [x] **Shareable opportunity links with clean metadata summaries** — BD people need to share with capture teams. Add OG meta tags for opportunity pages so links preview nicely in Slack/Teams/email
- [ ] **Interested Vendors List (IVL) data** — if SAM.gov API exposes IVL information, show vendor interest count and interested companies. Useful for gauging competition and identifying teaming partners

---

## 4. Save Opportunities

Saved searches already exist (localStorage, Advanced Search page). Individual opportunity bookmarking does not exist yet.

- [ ] Save/bookmark individual opportunities (requires auth)
- [ ] Free tier: 10 saved, paid: unlimited
- [ ] Notes field (editable inline)
- [ ] Filter: Active/Archived
- [ ] Export to CSV (paid feature)
- [ ] Server-side persistence (replace current localStorage saved searches)

---

## 5. Stripe Integration

- [ ] Checkout flow ($9/month or $79/year)
- [ ] Webhook handling (subscription events)
- [ ] Subscription management (upgrade, downgrade, cancel)
- [ ] Signup agreement text ("By creating an account, you agree to...")
- [ ] Free vs. Pro feature gating
