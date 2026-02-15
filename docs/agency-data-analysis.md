# Agency Data Analysis

Current state of the 8 agency-related columns in the `opportunities` table.

**Dataset**: 26,857 opportunities (as of 2026-02-14)

---

## Column Fill Rates

```sql
SELECT
  COUNT(*) as total,
  COUNT(*) FILTER (WHERE department != '') as dept_filled,
  COUNT(*) FILTER (WHERE sub_tier != '') as subtier_filled,
  COUNT(*) FILTER (WHERE office != '') as office_filled,
  COUNT(*) FILTER (WHERE full_parent_path_name != '') as fppn_filled,
  COUNT(*) FILTER (WHERE full_parent_path_code != '') as fppc_filled,
  COUNT(*) FILTER (WHERE cgac != '') as cgac_filled,
  COUNT(*) FILTER (WHERE fpds_code != '') as fpds_filled,
  COUNT(*) FILTER (WHERE aac_code != '') as aac_filled
FROM opportunities;
```

| Column | Filled | Fill Rate | Distinct Values |
|--------|--------|-----------|-----------------|
| `department` | 26,857 | **100%** | 48 |
| `sub_tier` | 26,793 | **99.8%** | 125 |
| `office` | 26,847 | **99.96%** | 1,256 |
| `full_parent_path_name` | 26,857 | **100%** | 1,260 |
| `full_parent_path_code` | 0 | **0%** | 0 |
| `cgac` | 26,857 | **100%** | 48 |
| `fpds_code` | 26,857 | **100%** | 132 |
| `aac_code` | 26,544 | **98.8%** | 1,311 |

**Key finding**: `full_parent_path_code` is completely empty across all rows — but **the API does return it** (see [API Sample Analysis](#api-sample-analysis) below). The CSV bulk download doesn't include this field, and our CSV ingestion path never sets it. The other 7 columns are near-100% filled.

---

## Column Roles

| Column | Level | Description |
|--------|-------|-------------|
| `department` | 1 (top) | Cabinet department or independent agency name |
| `sub_tier` | 2 | Sub-agency or major command (e.g., "DEPT OF THE NAVY") |
| `office` | 3 | Contracting office (e.g., "DLA LAND AND MARITIME") |
| `cgac` | 1 | 3-digit numeric code for the department (1:1 with `department`) |
| `fpds_code` | 2 | 4-char alphanumeric code for sub-tier (1:1 with `sub_tier`) |
| `aac_code` | 3 | Activity Address Code for the office (many offices can share same aac) |
| `full_parent_path_name` | all | Dot-separated hierarchy: `dept.subtier.office` |
| `full_parent_path_code` | all | Should be dot-separated code path — **always empty** |

---

## Department Distribution (Top 15)

```sql
SELECT department, COUNT(*) as count,
  ROUND(COUNT(*)::numeric / 26857 * 100, 1) as pct
FROM opportunities WHERE department != ''
GROUP BY department ORDER BY count DESC LIMIT 15;
```

| Department | Count | % |
|-----------|-------|---|
| DEPT OF DEFENSE | 19,125 | 71.2% |
| VETERANS AFFAIRS, DEPARTMENT OF | 2,665 | 9.9% |
| HOMELAND SECURITY, DEPARTMENT OF | 931 | 3.5% |
| INTERIOR, DEPARTMENT OF THE | 928 | 3.5% |
| AGRICULTURE, DEPARTMENT OF | 615 | 2.3% |
| HEALTH AND HUMAN SERVICES, DEPARTMENT OF | 405 | 1.5% |
| STATE, DEPARTMENT OF | 386 | 1.4% |
| JUSTICE, DEPARTMENT OF | 358 | 1.3% |
| GENERAL SERVICES ADMINISTRATION | 273 | 1.0% |
| ENERGY, DEPARTMENT OF | 256 | 1.0% |
| TRANSPORTATION, DEPARTMENT OF | 217 | 0.8% |
| COMMERCE, DEPARTMENT OF | 194 | 0.7% |
| NATIONAL AERONAUTICS AND SPACE ADMINISTRATION | 174 | 0.6% |
| TREASURY, DEPARTMENT OF THE | 67 | 0.2% |
| ENVIRONMENTAL PROTECTION AGENCY | 57 | 0.2% |

DoD alone accounts for **71%** of all opportunities. The top 5 departments cover **90%**.

---

## Sub-Tier Distribution (Top 20)

```sql
SELECT sub_tier, COUNT(*) as count,
  ROUND(COUNT(*)::numeric / 26857 * 100, 1) as pct
FROM opportunities WHERE sub_tier != ''
GROUP BY sub_tier ORDER BY count DESC LIMIT 20;
```

| Sub-Tier | Count | % |
|----------|-------|---|
| DEFENSE LOGISTICS AGENCY | 10,611 | 39.5% |
| DEPT OF THE NAVY | 3,832 | 14.3% |
| DEPT OF THE ARMY | 2,744 | 10.2% |
| VETERANS AFFAIRS, DEPARTMENT OF | 2,659 | 9.9% |
| DEPT OF THE AIR FORCE | 1,457 | 5.4% |
| US COAST GUARD | 724 | 2.7% |
| STATE, DEPARTMENT OF | 386 | 1.4% |
| FOREST SERVICE | 349 | 1.3% |
| ENERGY, DEPARTMENT OF | 252 | 0.9% |
| FEDERAL PRISON SYSTEM / BUREAU OF PRISONS | 246 | 0.9% |
| NATIONAL PARK SERVICE | 231 | 0.9% |
| INDIAN HEALTH SERVICE | 225 | 0.8% |
| US FISH AND WILDLIFE SERVICE | 216 | 0.8% |
| NATIONAL AERONAUTICS AND SPACE ADMINISTRATION | 169 | 0.6% |
| FEDERAL ACQUISITION SERVICE | 165 | 0.6% |
| BUREAU OF RECLAMATION | 138 | 0.5% |
| AGRICULTURAL RESEARCH SERVICE | 122 | 0.5% |
| BUREAU OF INDIAN AFFAIRS | 117 | 0.4% |
| FEDERAL HIGHWAY ADMINISTRATION | 109 | 0.4% |
| DEFENSE INFORMATION SYSTEMS AGENCY (DISA) | 106 | 0.4% |

DLA alone is nearly **40%** of all opportunities.

---

## Office Distribution (Top 15)

```sql
SELECT office, COUNT(*) as count,
  ROUND(COUNT(*)::numeric / 26857 * 100, 1) as pct
FROM opportunities WHERE office != ''
GROUP BY office ORDER BY count DESC LIMIT 15;
```

| Office | Count | % |
|--------|-------|---|
| DLA LAND AND MARITIME | 5,263 | 19.6% |
| DLA AVIATION | 2,965 | 11.0% |
| NAVSUP WEAPON SYSTEMS SUPPORT MECH | 1,502 | 5.6% |
| DLA TROOP SUPPORT | 951 | 3.5% |
| NAVSUP WEAPON SYSTEMS SUPPORT | 405 | 1.5% |
| AVIATION LOGISTICS CENTER (ALC)(00038) | 258 | 1.0% |
| SPRMM1 DLA MECHANICSBURG | 239 | 0.9% |
| DLA AVIATION AT OKLAHOMA CITY, OK | 216 | 0.8% |
| DLA AVIATION AT PHILADELPHIA, PA | 214 | 0.8% |
| W6QM MICC-FT DRUM | 204 | 0.8% |
| DLA LAND WARREN | 186 | 0.7% |
| 250-NETWORK CONTRACT OFFICE 10 (36C250) | 180 | 0.7% |
| NAVFACSYSCOM MID-ATLANTIC | 166 | 0.6% |
| 242-NETWORK CONTRACT OFFICE 02 (36C242) | 151 | 0.6% |
| 246-NETWORK CONTRACTING OFFICE 6 (36C246) | 146 | 0.5% |

Office names are inconsistent — some include codes (W6QM, 36C250), location info, or abbreviations. 1,256 distinct values across 26,857 rows.

---

## Code Mappings

### CGAC → Department (1:1)

```sql
SELECT cgac, department, COUNT(*) as count
FROM opportunities WHERE cgac != ''
GROUP BY cgac, department ORDER BY count DESC LIMIT 15;
```

| CGAC | Department | Count |
|------|-----------|-------|
| 097 | DEPT OF DEFENSE | 11,105 |
| 017 | DEPT OF DEFENSE | 3,832 |
| 021 | DEPT OF DEFENSE | 2,744 |
| 036 | VETERANS AFFAIRS | 2,665 |
| 057 | DEPT OF DEFENSE | 1,444 |
| 070 | HOMELAND SECURITY | 931 |
| 014 | INTERIOR | 928 |
| 012 | AGRICULTURE | 615 |

Note: DoD uses **multiple CGAC codes** — 097 (DLA/defense agencies), 017 (Navy), 021 (Army), 057 (Air Force). CGAC is actually 1:1 with `sub_tier` for DoD, not with `department`.

### FPDS Code → Sub-Tier (1:1)

```sql
SELECT fpds_code, sub_tier, COUNT(*) as count
FROM opportunities WHERE fpds_code != '' AND sub_tier != ''
GROUP BY fpds_code, sub_tier ORDER BY count DESC LIMIT 15;
```

| FPDS Code | Sub-Tier | Count |
|-----------|----------|-------|
| 97AS | DEFENSE LOGISTICS AGENCY | 10,611 |
| 1700 | DEPT OF THE NAVY | 3,832 |
| 2100 | DEPT OF THE ARMY | 2,744 |
| 3600 | VETERANS AFFAIRS | 2,659 |
| 5700 | DEPT OF THE AIR FORCE | 1,457 |
| 7008 | US COAST GUARD | 724 |
| 1900 | STATE, DEPARTMENT OF | 386 |
| 12C2 | FOREST SERVICE | 349 |
| 8900 | ENERGY | 252 |
| 1540 | FEDERAL PRISON SYSTEM | 246 |
| 1443 | NATIONAL PARK SERVICE | 231 |
| 7527 | INDIAN HEALTH SERVICE | 225 |
| 1448 | US FISH AND WILDLIFE SERVICE | 216 |
| 8000 | NASA | 169 |
| 4732 | FEDERAL ACQUISITION SERVICE | 165 |

FPDS code is a clean 1:1 mapping to sub-tier name — a reliable lookup key.

---

## Hierarchy Path Analysis

### Path Levels

```sql
SELECT
  array_length(string_to_array(full_parent_path_name, '.'), 1) as levels,
  COUNT(*) as count
FROM opportunities WHERE full_parent_path_name != ''
GROUP BY levels ORDER BY count DESC;
```

| Levels | Count | % |
|--------|-------|---|
| 3 | 26,469 | 98.6% |
| 5 | 308 | 1.1% |
| 2 | 74 | 0.3% |
| 4 | 6 | 0.02% |

The standard pattern is **3 levels**: `department.sub_tier.office`.

### 2-Level Paths (74 rows — missing office)

```sql
SELECT full_parent_path_name, COUNT(*) as count
FROM opportunities
WHERE array_length(string_to_array(full_parent_path_name, '.'), 1) = 2
GROUP BY full_parent_path_name ORDER BY count DESC LIMIT 10;
```

| Path | Count |
|------|-------|
| DEPT OF DEFENSE.DEPT OF DEFENSE | 30 |
| FEDERAL DEPOSIT INSURANCE CORPORATION.FEDERAL DEPOSIT INSURANCE CORPORATION | 10 |
| GENERAL SERVICES ADMINISTRATION.GENERAL SERVICES ADMINISTRATION | 9 |
| HOMELAND SECURITY, DEPARTMENT OF.HOMELAND SECURITY, DEPARTMENT OF | 7 |
| VETERANS AFFAIRS, DEPARTMENT OF.VETERANS AFFAIRS, DEPARTMENT OF | 6 |
| NATIONAL AERONAUTICS AND SPACE ADMINISTRATION.NASA | 5 |

These are opportunities posted at the department/agency level without a specific office.

### 5-Level Paths (308 rows — extra depth)

```sql
SELECT full_parent_path_name, COUNT(*) as count
FROM opportunities
WHERE array_length(string_to_array(full_parent_path_name, '.'), 1) = 5
GROUP BY full_parent_path_name ORDER BY count DESC LIMIT 5;
```

| Path | Count |
|------|-------|
| STATE, DEPARTMENT OF.STATE, DEPARTMENT OF.U.S. EMBASSY PARIS | 16 |
| TRANSPORTATION, DEPARTMENT OF.OFFICE OF THE SECRETARY.6913G6 VOLPE NATL. TRANS. SYS CNTR | 14 |
| STATE, DEPARTMENT OF.STATE, DEPARTMENT OF.U.S. EMBASSY LONDON | 12 |
| HOMELAND SECURITY, DEPARTMENT OF.US SECRET SERVICE.U. S. SECRET SERVICE | 12 |
| JUSTICE, DEPARTMENT OF.US MARSHALS SERVICE.U.S. DEPT OF JUSTICE, USMS | 9 |

**Originally assumed** these were 3-level paths where periods in the office name caused false splits. However, the API analysis below reveals these are actually legitimate multi-level hierarchies — the issue is that our CSV ingestion only captures 3 levels (dept, sub-tier, office) and constructs `full_parent_path_name` from those 3 fields, losing any intermediate levels.

---

## Independent Agencies (department = sub_tier)

```sql
SELECT COUNT(*) FILTER (WHERE department = sub_tier AND department != '') as count,
  ROUND(COUNT(*) FILTER (WHERE department = sub_tier AND department != '')::numeric / COUNT(*) * 100, 1) as pct
FROM opportunities;
```

**3,708 rows (13.8%)** have `department = sub_tier`. These are independent agencies that don't have a parent department:

| Agency | Count |
|--------|-------|
| VETERANS AFFAIRS, DEPARTMENT OF | 2,659 |
| STATE, DEPARTMENT OF | 386 |
| ENERGY, DEPARTMENT OF | 252 |
| NATIONAL AERONAUTICS AND SPACE ADMINISTRATION | 169 |
| ENVIRONMENTAL PROTECTION AGENCY | 57 |
| LIBRARY OF CONGRESS | 31 |
| UNITED STATES GOVERNMENT PUBLISHING OFFICE | 23 |
| SOCIAL SECURITY ADMINISTRATION | 18 |
| POSTAL SERVICE | 13 |
| FEDERAL DEPOSIT INSURANCE CORPORATION | 10 |

---

## DoD Sub-Tier Breakdown (22 organizations)

```sql
SELECT DISTINCT sub_tier
FROM opportunities WHERE department = 'DEPT OF DEFENSE' AND sub_tier != ''
ORDER BY sub_tier;
```

| Sub-Tier |
|----------|
| DEFENSE ADVANCED RESEARCH PROJECTS AGENCY (DARPA) |
| DEFENSE COMMISSARY AGENCY (DECA) |
| DEFENSE COUNTERINTELLIGENCE AND SECURITY AGENCY (DCSA) |
| DEFENSE FINANCE AND ACCOUNTING SERVICE (DFAS) |
| DEFENSE HEALTH AGENCY (DHA) |
| DEFENSE HUMAN RESOURCES ACTIVITY |
| DEFENSE INFORMATION SYSTEMS AGENCY (DISA) |
| DEFENSE INTELLIGENCE AGENCY (DIA) |
| DEFENSE LOGISTICS AGENCY |
| DEFENSE MEDIA ACTIVITY (DMA) |
| DEFENSE MICROELECTRONICS ACTIVITY (DMEA) |
| DEFENSE THREAT REDUCTION AGENCY (DTRA) |
| DEPT OF DEFENSE EDUCATION ACTIVITY (DODEA) |
| DEPT OF THE AIR FORCE |
| DEPT OF THE ARMY |
| DEPT OF THE NAVY |
| MISSILE DEFENSE AGENCY (MDA) |
| NATIONAL GEOSPATIAL-INTELLIGENCE AGENCY (NGA) |
| UNIFORMED SERVICES UNIVERSITY OF THE HEALTH SCIENCES (USUHS) |
| US SPECIAL OPERATIONS COMMAND (USSOCOM) |
| US TRANSPORTATION COMMAND (USTRANSCOM) |
| WASHINGTON HEADQUARTERS SERVICES (WHS) |

---

## API Sample Analysis

Source: `opportunities.json` in project root — 100 opportunities from a single API page (of 1,353 total).

### `fullParentPathCode` IS Populated in the API

The API returns `fullParentPathCode` on **100% of opportunities**. Our database has 0% because the CSV bulk download doesn't include this column, and our CSV ingestion (`samgov/csv.go`) never sets `FullParentPathCode`.

The API ingestion path (`samgov/api.go:245`) does map this field, but we ingest via CSV, not the API.

### The Hierarchy Is Deeper Than 3 Levels

The database shows mostly 3-level paths because the CSV only provides `Department/Ind.Agency`, `Sub-Tier`, and `Office` — which we join with `.` to make `full_parent_path_name`. But the API reveals the real hierarchy goes **up to 7 levels deep**:

| Levels | API Sample | DB (CSV-constructed) |
|--------|-----------|---------------------|
| 3 | 35 (35%) | 26,469 (98.6%) |
| 4 | 6 (6%) | 6 (0.02%) |
| 5 | 20 (20%) | 308 (1.1%) |
| 6 | 32 (32%) | 74 (0.3%) |
| 7 | 7 (7%) | — |

Only 35% of API results have the flat 3-level structure. **65% have intermediate organizational layers** between sub-tier and office.

### Path Structure

The code path follows the pattern: `CGAC.FPDS.[intermediate abbreviations...].AAC_CODE`

The last segment is **always 6 characters** (the AAC code).

### Examples by Depth

**3 levels** (flat agencies — VA, Energy, Interior, etc.):
```
Name: INTERIOR, DEPARTMENT OF THE.US FISH AND WILDLIFE SERVICE.FWS SAT TEAM 3
Code: 014.1448.140FS3
```

**4 levels** (one intermediate layer):
```
Name: DEPT OF DEFENSE.DEFENSE LOGISTICS AGENCY.TROOP SUPPORT SUBSISTENCE.DLA TROOP SUPPORT
Code: 097.97AS.DLA TRP SUPT SUB.SPE300
```

**5 levels** (DLA, NAVSEA subdivisions):
```
Name: DEPT OF DEFENSE.DEFENSE LOGISTICS AGENCY.DLA AVIATION.DLA AVIATION OKLAHOMA CITY.DLA AVIATION AT OKLAHOMA CITY, OK
Code: 097.97AS.DLA AVIATION.DLA AV OKLAHOMA CITY.SPRTA1
```

**6 levels** (Navy, Army, Air Force commands):
```
Name: DEPT OF DEFENSE.DEPT OF THE NAVY.NAVSUP.NAVSUP GLOBAL LOGISTICS SUPPORT.NAVSUP FLC SAN DIEGO.NAVSUP FLT LOG CTR  SAN DIEGO
Code: 017.1700.NAVSUP.NAVSUP GLS.NAVSUP FLC SAN DIEGO.N00244
```

**7 levels** (Army MICC chain):
```
Name: DEPT OF DEFENSE.DEPT OF THE ARMY.AMC.ACC.MISSION INSTALLATION CONTRACTING COMMAND.419TH CSB.W6QM MICC-FT DRUM
Code: 021.2100.AMC.ACC.MICC.419TH CSB.W911S2
```

### Intermediate Org Layers

The levels between sub-tier and office represent major commands and subcommands. Most common in the 100-record sample:

| Intermediate Org | Count |
|-----------------|-------|
| NAVSUP | 21 |
| NAVSUP WEAPON SYSTEMS SUPPORT | 18 |
| NAVSUP WSS MECHANICSBURG | 15 |
| AMC (Army Materiel Command) | 7 |
| ACC (Army Contracting Command) | 7 |
| DLA AVIATION | 6 |
| AIR FORCE MATERIEL COMMAND | 5 |
| US ARMY CORPS OF ENGINEERS | 4 |
| NAVSEA | 4 |
| NAVFAC | 3 |

### Code Path Contains Abbreviations

The `fullParentPathCode` intermediate levels use **abbreviations** rather than formal codes:

| Name | Code |
|------|------|
| NAVSUP GLOBAL LOGISTICS SUPPORT | NAVSUP GLS |
| DLA MARITIME SHIPYARDS | DLA MARITIME SHIPYDS |
| NAVSUP WEAPON SYSTEMS SUPPORT | NAVSUP WSS |
| DLA AVIATION OKLAHOMA CITY | DLA AV OKLAHOMA CITY |
| NATIONAL GUARD BUREAU | NGB |
| AIR FORCE MATERIEL COMMAND | AFMC |
| AIR COMBAT COMMAND | ACC |
| MISSION INSTALLATION CONTRACTING COMMAND | MICC |

These abbreviations are widely used by contractors and would be excellent search targets.

### CSV vs API Field Comparison

| Data Point | CSV Columns | API Fields |
|-----------|-------------|------------|
| Department name | `Department/Ind.Agency` | `fullParentPathName` (part 1) |
| Sub-tier name | `Sub-Tier` | `fullParentPathName` (part 2) |
| Office name | `Office` | `fullParentPathName` (last part) |
| Full hierarchy path | — (we construct it) | `fullParentPathName` (native, 3-7 levels) |
| Full code path | — (not in CSV) | `fullParentPathCode` (CGAC.FPDS.abbrevs.AAC) |
| CGAC code | `CGAC` | `fullParentPathCode` (part 1) |
| FPDS code | `FPDS Code` | `fullParentPathCode` (part 2) |
| AAC code | `AAC Code` | `fullParentPathCode` (last part) |
| Org type | `OrganizationType` | `organizationType` (always "OFFICE") |
| Office address | `City`, `State`, `ZipCode`, `CountryCode` | `officeAddress` (object) |

---

## Summary & Implications for Agency Search

1. **7 of 8 columns are reliably populated in the DB** — `full_parent_path_code` is empty because the CSV doesn't include it, but the API does return it.
2. **The hierarchy is deeper than we store** — the API reveals 3-7 levels with intermediate commands (NAVSUP, AMC, NAVFAC, etc.), but our CSV ingestion flattens everything to 3 levels (dept.subtier.office).
3. **`fpds_code` → `sub_tier`** is the cleanest lookup key (1:1, 100% filled).
4. **`cgac`** maps to sub-tier level for DoD (not department), making it less useful as a department-level key.
5. **Office names are messy** — abbreviations, codes embedded in names, inconsistent formatting. The `aac_code` is more reliable for office-level identification.
6. **The code path contains useful abbreviations** (NAVSUP, AFMC, MICC, NGB, ACC) that contractors commonly search by — these are currently not captured at all.
7. **48 departments, 125 sub-tiers, 1,256 offices** — the hierarchy is manageable but needs an alias/abbreviation layer (users will search for "DLA" not "DEFENSE LOGISTICS AGENCY").
8. **DoD dominates** — 71% of opportunities. The 22 DoD sub-tiers need special attention in any search UI since that's where most users will be filtering.
9. **To capture the full hierarchy**, we'd either need to switch to API ingestion (rate-limited) or supplement the CSV data with a one-time hierarchy build from the Federal Hierarchy API or from `fullParentPathCode`/`fullParentPathName` data collected via the opportunities API.
