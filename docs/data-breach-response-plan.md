# Data Breach Response Plan

**Last reviewed:** February 2026
**Data Controller:** András Hinkel (sole proprietor, Hungary)
**Supervisory Authority:** NAIH — 1055 Budapest, Falk Miksa utca 9-11, ugyfelszolgalat@naih.hu, https://naih.hu

---

## 1. Detection

A breach may be discovered through:

- AWS billing/cost anomaly alerts
- App Runner error rate spikes (CloudWatch)
- Cloudflare WAF security events or bot alerts
- Neon database access notifications
- WorkOS security notifications
- User reports (compromised account, unexpected data)
- Manual log review

**If you suspect a breach, start the clock. You have 72 hours from awareness to notify NAIH if required.**

---

## 2. Immediate Response (first hour)

- [ ] **Contain** — Revoke compromised credentials, rotate API keys/secrets, block malicious IPs via Cloudflare
- [ ] **Preserve evidence** — Screenshot logs, export CloudWatch/Cloudflare events, note timestamps. Do not delete anything.
- [ ] **Assess scope** — Answer these questions:
  - What data was accessed? (emails, names, saved searches, search events, payment data)
  - How many users are affected?
  - How did the breach occur? (leaked credentials, vulnerability, third-party compromise)
  - Is the breach ongoing or contained?

---

## 3. Risk Assessment

Determine if NAIH notification is required. **Notify unless the breach is unlikely to result in a risk to users' rights and freedoms.**

| Scenario | Risk Level | Notify NAIH? | Notify Users? |
|----------|-----------|-------------|--------------|
| Leaked API key, no data accessed | Low | No (document only) | No |
| Unauthorized DB read of emails/names | Medium | Yes | Likely yes |
| Full DB dump including search history | High | Yes | Yes |
| WorkOS breach (auth compromise) | High | Yes (+ coordinate with WorkOS) | Yes |
| Stripe breach (payment data) | High | Yes (+ coordinate with Stripe) | Yes |

**When in doubt, notify.** Over-reporting is not penalized; under-reporting is.

---

## 4. NAIH Notification (within 72 hours)

Submit breach notification to NAIH: https://naih.hu/adatvedelmi-incidens-bejelentes

Include:
- Nature of the breach (what happened)
- Categories and approximate number of affected users
- Data categories affected (email, name, search history, etc.)
- Likely consequences
- Measures taken or proposed to address the breach
- Contact details (privacy@govtrove.com)

If you don't have full details within 72 hours, submit what you know and provide updates as information becomes available. GDPR allows phased notification.

---

## 5. User Notification (without undue delay, if high risk)

Email affected users with:
- Plain-language description of what happened
- What data was involved
- What you've done to fix it
- What they should do (change passwords, monitor accounts)
- Contact: privacy@govtrove.com

Template:

> Subject: Security Notice — GovTrove
>
> We're writing to inform you of a security incident that may have affected your GovTrove account.
>
> **What happened:** [brief description]
>
> **What data was involved:** [email, name, saved searches, etc.]
>
> **What we've done:** [containment and remediation steps]
>
> **What you should do:** [change password, review account, etc.]
>
> We take the security of your data seriously and apologize for this incident. If you have questions, contact us at privacy@govtrove.com.

---

## 6. Remediation

- [ ] Patch the vulnerability or close the attack vector
- [ ] Rotate all potentially compromised secrets (DATABASE_URL, WORKOS_API_KEY, WORKOS_CLIENT_ID, SAM API key)
- [ ] Force sign-out affected users (WorkOS session revocation)
- [ ] Review access logs for full scope of unauthorized access
- [ ] Update security measures to prevent recurrence

### Secret Rotation Locations

| Secret | Stored In | Update Procedure |
|--------|-----------|-----------------|
| DATABASE_URL | AWS Secrets Manager + Neon | Rotate in Neon, update secret in AWS, redeploy API |
| WORKOS_API_KEY | AWS Secrets Manager | Rotate in WorkOS dashboard, update secret in AWS, redeploy API |
| WORKOS_CLIENT_ID | AWS Secrets Manager | WorkOS dashboard, update secret, redeploy |
| SAM_API_KEY | AWS Secrets Manager | Request new key at sam.gov, update secret, redeploy pipeline |

---

## 7. Documentation

Record the following regardless of whether NAIH was notified (GDPR Art. 33(5)):

- Date and time breach was discovered
- Date and time breach was contained
- Nature of the breach
- Data and users affected
- Consequences (known and likely)
- Actions taken
- Decision on NAIH notification (and reasoning if not notified)
- Decision on user notification (and reasoning if not notified)

Store this record in a private location (not in the git repo). Retain for at least 5 years.
