# SAM.gov API Rate Limits

## Rate Limits by Account Type

| User Type | API Key Type | Daily Limit |
|-----------|--------------|-------------|
| Non-federal (no role) | Personal | **10 requests/day** |
| Non-federal (with role) | Personal | **1,000 requests/day** |
| Federal user | Personal | **1,000 requests/day** |
| Non-federal system account | System | **1,000 requests/day** |
| Federal system account | System | **10,000 requests/day** |

**Hourly limit (via api.data.gov):** 1,000 requests/hour on a rolling basis

## API Key Expiration

- **Keys expire every 90 days**
- Email notifications sent starting 15 days before expiration
- New key is auto-generated before current key expires

## When Limits Are Exceeded

- HTTP **429 (Too Many Requests)** response
- API key is **temporarily blocked**
- Block lifts automatically after **1 hour** (rolling window)
- Example: 500 requests at 10:15 + 500 at 10:25 = blocked until 11:15

## Response Headers

Every API response includes:
- `X-RateLimit-Limit`: Your maximum request allowance
- `X-RateLimit-Remaining`: Remaining requests available

## Requesting Higher Limits

- Only **federal system accounts** can request rate limit increases
- Handled case-by-case based on business justification
- Can be revoked if straining IAE systems or underutilized

## Implications for OpScout

With a standard personal API key (1,000/day):
- **Search API**: At 1,000 records/page, can fetch ~1M opportunities in 1,000 requests
- **Description API**: Limited to 1,000 descriptions/day (333ms delay = ~5.5 minutes for 1,000)
- **CSV download**: Does NOT count against API limits (no API key needed)

### Recommendations

1. Always fetch CSV first (unlimited)
2. Use API only for records not in CSV
3. Implement exponential backoff on 429 errors
4. Cache `X-RateLimit-Remaining` to avoid hitting limits
5. Consider requesting a system account if federal partnership possible

## What is a "Role"?

A "role" refers to data access permissions assigned to your SAM.gov account:

- **No role** = Basic SAM.gov account → 10 requests/day
- **With role** = Account with granted data access permissions (e.g., "Read Public") → 1,000 requests/day

To get a role, you typically need to register an entity in SAM.gov or be granted permissions by an administrator.

## Sources

- [SAM.gov Get Opportunities Public API](https://open.gsa.gov/api/get-opportunities-public-api/)
- [SAM.gov Entity Management API](https://open.gsa.gov/api/entity-api/)
- [api.data.gov Developer Manual](https://api.data.gov/docs/developer-manual/)
- [GovCon API SAM.gov Guide](https://govconapi.com/sam-gov-api-guide)
- [SAM.gov Role Management Guide](https://dodprocurementtoolbox.com/uploads/SAM_Role_Management_Guide_v1_283960a128.pdf)
