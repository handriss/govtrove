# EU-US Data Transfer Safeguards

**Verified:** February 23, 2026
**Data Controller:** András Hinkel (sole proprietor, Hungary)

Per GDPR Chapter V, personal data transfers to countries outside the EU/EEA require appropriate safeguards. This document records the legal mechanism covering each of our US-based processors.

## Processor Verification

| Processor | Purpose | Location | DPF Certified? | Safeguard | DPA |
|-----------|---------|----------|---------------|-----------|-----|
| AWS | Hosting, infrastructure | US | Yes ([#5776](https://www.dataprivacyframework.gov/participant/5776)) | DPF | [AWS DPA](https://docs.aws.amazon.com/whitepapers/latest/navigating-gdpr-compliance/aws-data-processing-addendum-dpa.html) |
| Stripe | Payment processing | US | Yes ([#6436](https://www.dataprivacyframework.gov/participant/6436)) | DPF | [Stripe DPA](https://stripe.com/legal/dpa) |
| Neon | Database hosting | US | Yes | DPF + SCCs | [Neon DPA](https://neon.com/dpa) |
| WorkOS | Authentication | US | No | SCCs (in DPA) | [WorkOS DPA](https://workos.com/legal/data-processing-addendum) |

## Notes

- **DPF** = EU-US Data Privacy Framework. Company self-certifies with US Dept. of Commerce. EU recognizes this as adequate protection per adequacy decision of July 10, 2023.
- **SCCs** = Standard Contractual Clauses. Pre-approved contract templates from the European Commission (Decision 2021/914). Valid alternative when DPF is not available.
- **WorkOS** is not DPF-certified but includes SCCs in their Data Processing Addendum, which is an equally valid GDPR transfer mechanism.

## Privacy Policy Alignment

Our Privacy Policy (Section 5) states: "Our US-based processors are certified under the EU-U.S. Data Privacy Framework (DPF), and/or we rely on Standard Contractual Clauses (SCCs)." This is accurate for all processors listed above.

## Review Schedule

Re-verify annually or when adding a new processor. DPF certifications can lapse — check https://www.dataprivacyframework.gov/list if in doubt.
