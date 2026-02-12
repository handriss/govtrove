# GovTrove — Next Epic Backlog

## Operational Items

- [ ] Cloudflare WAF custom rules for API (5 free rules available)
- [ ] CloudFront access logs (for traffic visibility)
- [ ] Mobile-friendly results table — current `<table>` with 8 columns is unusable on phones. Switch to card layout on small screens or hide non-essential columns (agency, NAICS, state).
- [ ] Display contact info on opportunity detail page — data exists in `opportunity_contacts` table but is not fetched or rendered

---

## 1. Authentication (WorkOS AuthKit)

WorkOS AuthKit — free up to 1M MAU, official Go SDK, hosted login UI, built-in social login, MFA, passkeys, RBAC. Replaces any prior Neon Auth plans. Drop existing Neon Auth tables when implementing.

### Phase 1
- [ ] "Microsofts sso" (primary, most prominent) — GovCon small businesses overwhelmingly use Microsoft 365 due to CMMC/NIST compliance
- [ ] "Continue with Google" (secondary) — catches early-stage contractors using Google Workspace or personal Gmail
- [ ] Email + Password (fallback) — essential for orgs that restrict OAuth. WorkOS handles password strength, leak detection, secure hashing

### Phase 2
- [ ] LinkedIn SSO — GovCon networking is heavily LinkedIn-centered
- [ ] Magic Auth (email OTP) — more reliable than magic links because enterprise email scanners click links and invalidate tokens

### Implementation Notes
- Backend: WorkOS Go SDK (`github.com/workos/workos-go`), receive auth code from AuthKit redirect, exchange for session, store WorkOS user ID -> GovTrove user ID mapping in Neon PostgreSQL
- Frontend: Login button redirects to WorkOS AuthKit hosted page. No custom login/signup forms needed
- Cost: $0 (within 1M MAU free tier). Enterprise SSO upgrade path at $125/connection/month
- Security: SOC 2 certified, GDPR/CCPA compliant. GovTrove never sees OAuth tokens or raw passwords

---

## 2. Landing Page Messaging Revision

Positioning principles to apply when revising the landing page:

- [x] **Metadata-first philosophy as core positioning** — "You don't read a 50-page SOW to decide if an opportunity is worth your time. Neither should your search tool." Reframes what competitors treat as a limitation into a deliberate product philosophy
- [x] **Messaging hierarchy**: speed -> simplicity -> intelligence (avoid leading with "AI")
- [x] **Transparent pricing from day one** — show pricing on the landing page, no "contact sales" gates. Free vs. Pro comparison table. In a market full of "request a demo" opacity, transparent pricing is a differentiator
- [x] **SEO targeting**: "SAM.gov search alternative", "find government contracts free", "SAM.gov frustrations"
- [x] **Scam awareness & trust positioning** — predatory ecosystem charges $600-$1,500 for SAM.gov registration (which is free). GSA OIG identified 400+ fraud attempts since 2021. Content/blog opportunity: "SAM.gov Registration Is Free — Here's How" drives SEO traffic and establishes credibility

---

## 3. Search UX Improvements

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

## 4. Opportunity Detail Page

Most metadata is already displayed: solicitation number, notice type, dates (posted/response/archive), set-aside with color-coded badges, NAICS codes, agency/department hierarchy, place of performance, award info (number, amount, date, awardee), attachments/resources with links, "View on SAM.gov" link, expandable description.

Still to build:

- [ ] **Contact info display** — data exists in `opportunity_contacts` table (name, email, phone, type) but is not fetched or rendered on the detail page
- [ ] **Timeline/history of amendments** — show amendment history when available
- [ ] **Shareable opportunity links with clean metadata summaries** — BD people need to share with capture teams. Add OG meta tags for opportunity pages so links preview nicely in Slack/Teams/email
- [ ] **Interested Vendors List (IVL) data** — if SAM.gov API exposes IVL information, show vendor interest count and interested companies. Useful for gauging competition and identifying teaming partners

---

## 5. Save Opportunities

Saved searches already exist (localStorage, Advanced Search page). Individual opportunity bookmarking does not exist yet.

- [ ] Save/bookmark individual opportunities (requires auth)
- [ ] Free tier: 10 saved, paid: unlimited
- [ ] Notes field (editable inline)
- [ ] Filter: Active/Archived
- [ ] Export to CSV (paid feature)
- [ ] Server-side persistence (replace current localStorage saved searches)

---

## 6. Stripe Integration

Blocked on auth (section 1).

- [ ] Checkout flow ($9/month or $79/year)
- [ ] Webhook handling (subscription events)
- [ ] Subscription management (upgrade, downgrade, cancel)
- [ ] Signup agreement text ("By creating an account, you agree to...")
- [ ] Free vs. Pro feature gating
