# Claude Code Guidelines for OpScout

## Compliance-First Principle

Every technical decision, architecture choice, and implementation detail must be evaluated for compliance. This is not just about following rules — it's about respecting the intent behind the rules and building a business that the government contracting community can trust.

### When a Compliance Question Arises

1. **Flag it immediately** — don't assume it's fine
2. **Explain the specific rule or policy** that applies
3. **Explain the spirit/intent** behind the regulation — why does this rule exist? What behavior is it trying to prevent? What outcome is it trying to protect?
4. **Assess the proposed approach** against both the letter AND spirit of the rule. Something that is technically not prohibited but clearly violates the intent is NOT compliant.
5. **Provide a compliant alternative** if the original approach has issues
6. **Rate the compliance risk**: Green (clearly allowed), Yellow (gray area — proceed with caution), Red (clearly prohibited or violates intent)

### The "Spirit of the Law" Standard

When evaluating any approach, apply this test:
- Would GSA (the agency running SAM.gov) be comfortable if they knew exactly what we're doing?
- Would our approach still work if every SAM.gov user did the same thing?
- Are we adding value to the ecosystem, or are we extracting/exploiting it?
- If a GSA employee reviewed our system's behavior, would they see a legitimate business tool or an abusive bot?

If the answer to any of these is uncomfortable, find a different approach.

### The "Alternative Approach" Standard

When I propose an alternative way of doing something, evaluate:
- Is this genuinely compliant, or is it a loophole/technicality?
- Does it respect the underlying purpose of the restriction?
- Would this approach survive scrutiny if SAM.gov/GSA reviewed it?
- Am I trying to get around a rule, or am I solving the same problem through a legitimately different path?

Being clever about circumventing restrictions is not the same as finding a compliant solution. Always distinguish between the two.

---

## Known SAM.gov Policies (from Terms of Use)

### Explicitly Prohibited
- **Web scraping is prohibited**: "Automated data gathering, web scraping tools are prohibited and, if detected, will result in the associated account(s) being denied access to SAM.gov via Login.gov."
- **D&B data restrictions**: D&B data may not be used for commercial, resale, or marketing purposes. D&B Open Data may not be shared in bulk.
- **Exceeding authorization**: "My system will not attempt to access areas of SAM my system is not authorized to access."
- **Malicious activity**: No submissions of malicious files, viruses, or harmful code.

### Explicitly Allowed
- **Public API access**: SAM.gov provides official REST APIs for accessing public data — this is the intended method for programmatic access.
- **Personal API keys**: Any registered user can get a personal API key for accessing public data.
- **System accounts**: For higher volume needs, non-federal users can request system accounts with higher rate limits.
- **Public data reuse**: Contract opportunity data (public notices) is government-produced public information.
- **Data extracts**: SAM.gov provides bulk data downloads through their Data Services.

### Gray Areas Requiring Caution
- **Attachment downloads at scale**: The API provides resourceLinks for attachments. Downloading them individually through the API appears allowed; mass automated downloading from the website may trigger blocks. If datacenter IPs are blocked for attachment downloads, respect this — it likely exists to manage server load. Look for official bulk download alternatives first.
- **Caching/storing public data**: Storing public opportunity data appears allowed (it's public government information). However, always include attribution and link back to SAM.gov.
- **Rate limits**: Default limits exist for a reason. Stay well within them. If you need more, apply for a system account through the proper channels rather than using multiple API keys or other workarounds.
- **Reselling data**: The opportunity data itself is public, but some entity data (especially D&B data) has restrictions. Be careful about which data you store and redistribute.

---

## Technical Context

- **App name**: OpScout
- **Stack**: Go backend, React frontend, Neon PostgreSQL, AWS (Fargate, EventBridge, S3, CloudFront), Terraform
- **Data source**: SAM.gov public API (contract opportunities)
- **Business model**: Freemium SaaS ($9/mo or $79/year paid tier)
- **Target users**: Small businesses pursuing DoD contracts
- **Data volume**: ~500-2,000 new opportunities/day, ~100K/year

## General Development Principles

- Prefer simple, cost-effective solutions
- Serverless/pay-per-use where possible
- Keep infrastructure costs under $50/month for MVP
- Use official APIs — never scraping unless explicitly permitted
- When in doubt about data rights, link to SAM.gov rather than cache/store
- Attribute SAM.gov as the data source wherever data is displayed
- Include links back to the original SAM.gov opportunity page
- Respect rate limits with generous margins (don't push to the edge)

---

## AWS Profile

**Always use the `opscout` AWS profile** for all AWS CLI and Terraform commands in this project.

```bash
--profile opscout
```

The Makefile already has `AWS_PROFILE := opscout` configured.

## Protected Files

**Never delete, overwrite, or modify these files:**
- `.env` - contains secrets and API keys
- `.env.local`, `.env.production`, or any `.env.*` files (except `.env.example`)
- Any file containing credentials or secrets

These files should only be created or edited by the user directly.

## Code Style

### Comments

Comments are for intent, not narration.
Write comments only when explaining why something exists (constraints, tradeoffs, landmines, or intentional ugliness).
If a comment explains what the code does, rewrite the code instead.
Assume comments rot; structure and names must carry meaning.
