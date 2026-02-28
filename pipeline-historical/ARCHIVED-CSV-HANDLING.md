# Archived CSV Handling — Current State

This document catalogs every place in the codebase that deals with archived CSV records, what each piece does, and how they connect.

---

## Overview

Archived CSV records flow through the pipeline in two roles:

1. **Data source** — archived records are parsed, converted to opportunities, and upserted into the `opportunities` table (with `active = false`).
2. **Disappearance resolver** — when a notice vanishes from the active CSV, the system checks if it appeared in the archived CSV. If yes, the disappearance is marked as expected ("archived") rather than a glitch.

---

## Pipeline Flow

```
download-csvs Lambda
  ├── Downloads active CSV → S3
  └── Downloads archived CSVs (FY2025, FY2026) → S3
          │
          ▼
Step Functions Map (IngestFiles)
  ├── type="active"   → ingest-active Lambda
  └── type="archived" → ingest-archived Lambda   ◀── ARCHIVED-SPECIFIC
          │
          ▼
Both write to pipeline.snap_csv (same table, different run_ids)
          │
          ▼
Reconcile Lambda
  ├── loadCSVOpps(archivedRunIDs)   ◀── loads archived first
  ├── loadCSVOpps(activeRunID)      ◀── active overwrites same notice_id
  ├── UpsertOpportunities()         ◀── archived records get active=false
  ├── ResolveExpectedDisappearances(activeRunID, archivedRunID)  ◀── ARCHIVED-SPECIFIC
  └── MarkDisappearedInactive()     ◀── skips records resolved as "archived"
```

---

## 1. Download: `bulkcsv/sources.go`

Defines the archived CSV download URLs.

```go
// pipeline/internal/bulkcsv/sources.go

const (
    SourceTypeActive   SourceType = "active"
    SourceTypeArchived SourceType = "archived"
)

var Sources = []Source{
    {
        Key:      "active",
        Type:     SourceTypeActive,
        URL:      samgov.FullCSVURL,
        S3Prefix: "raw/csv",
    },
    {
        Key:           "archived_fy2025",
        Type:          SourceTypeArchived,
        URL:           "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/Archived%20Data/FY2025_archived_opportunities.csv",
        S3Prefix:      "raw/archived-csv/FY2025",
        UseRangeProbe: true,
    },
    {
        Key:           "archived_fy2026",
        Type:          SourceTypeArchived,
        URL:           "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/Archived%20Data/FY2026_archived_opportunities.csv",
        S3Prefix:      "raw/archived-csv/FY2026",
        UseRangeProbe: true,
    },
}
```

**Also referenced in:**
- `pipeline/internal/bulkcsv/bulkcsv.go:81` — `cfg.S3ArchiveEnabled` gates S3 upload of archived CSVs
- `pipeline/internal/bulkcsv/bulkcsv.go:176,217` — `archiveFile()` helper handles S3 upload
- `pipeline/internal/config/config.go:17` — `S3ArchiveEnabled bool` config flag

---

## 2. Step Functions routing: `step-functions.asl.json`

Routes files by type to the correct Lambda.

```json
// infra/terraform/step-functions.asl.json (lines 63-125)

"RouteByType": {
    "Type": "Choice",
    "Choices": [
        {
            "Variable": "$.file.type",
            "StringEquals": "active",
            "Next": "IngestActive"
        },
        {
            "Variable": "$.file.type",
            "StringEquals": "archived",
            "Next": "IngestArchived"
        }
    ],
    "Default": "IngestActive"
},

"IngestArchived": {
    "Type": "Task",
    "Resource": "arn:aws:states:::lambda:invoke",
    "Parameters": {
        "FunctionName": "${ingest_archived_arn}",
        "Payload.$": "$"
    },
    "ResultSelector": {
        "status.$": "$.Payload.status",
        "run_id.$": "$.Payload.run_id",
        "job_type.$": "$.Payload.job_type"
    },
    // ...retries...
    "End": true
}
```

---

## 3. Ingestion: `ingest-archived` Lambda

**Full file:** `pipeline/cmd/lambda/ingest-archived/main.go`

This Lambda:
- Downloads gzip CSV from S3
- Parses rows via `samgov.ParseCSVStream`
- Calls `BulkInsertSnapCSV()` to COPY into `pipeline.snap_csv`
- Does **NOT** run change detection (no `DetectChanges`, `DetectDisappearances`)
- Returns `{status, run_id, job_type: "ingest-archived"}`

Key constants:
```go
const (
    jobType   = "ingest-archived"
    batchSize = 5000
)
```

Core pipeline function:
```go
// pipeline/cmd/lambda/ingest-archived/main.go:181-231

func (h *Handler) runPipeline(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, s3Key string) (int, error) {
    // Idempotency: check if this file was already ingested
    bulkLog, err := h.Store.GetBulkCSVLogByS3Key(ctx, s3Key)
    // ...idempotency checks...

    out, err := h.S3.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(h.Bucket),
        Key:    aws.String(s3Key),
    })
    // ...

    recordCount, err := h.processStream(ctx, runID, snapshotDate, downloadID, out.Body)
    // ...
}
```

**Tests:** `pipeline/cmd/lambda/ingest-archived/handler_test.go` (14 specs)
**Suite:** `pipeline/cmd/lambda/ingest-archived/ingest_suite_test.go`

---

## 4. Reconcile Lambda — Loading archived opps

**File:** `pipeline/cmd/lambda/reconcile/main.go`

### 4a. Extracting archived run IDs from input (lines 125-138)

```go
var archivedRunIDs []uuid.UUID
for i, r := range input.IngestionResults {
    // ...status check...
    switch r.JobType {
    case "snapshot-csv":
        activeRunID = rid
    case "ingest-archived":
        archivedRunIDs = append(archivedRunIDs, rid)
    }
}
```

### 4b. Loading archived opps first (lines 158-172)

```go
// 3. Load CSV opps: archived first, then active (active wins for same notice_id)
csvOpps := make(map[string]reconcile.Opportunity)
var allDQEntries []database.DataQualityEntry

for _, archivedRunID := range archivedRunIDs {
    opps, dq, err := h.loadCSVOpps(ctx, archivedRunID, snapshotDate)
    if err != nil {
        return nil, fmt.Errorf("load archived %s: %w", archivedRunID, err)
    }
    allDQEntries = append(allDQEntries, dq...)
    for _, opp := range opps {
        csvOpps[opp.NoticeID] = opp
    }
    h.Logger.Info("archived CSV loaded", "run_id", archivedRunID, "count", len(opps))
}
```

Active CSV loads after and overwrites same keys — this is how "active wins" works.

### 4c. `loadCSVOpps` helper (lines 249-270)

```go
func (h *Handler) loadCSVOpps(ctx context.Context, runID uuid.UUID, snapshotDate time.Time) ([]reconcile.Opportunity, []database.DataQualityEntry, error) {
    rows, err := h.Store.GetSnapCSVRawData(ctx, runID)
    // ...
    for _, raw := range rows {
        opp, issues := reconcile.FromCSV(raw)
        opps = append(opps, opp)
        // ...collect DQ entries with Source: "csv"...
    }
    return opps, dqEntries, nil
}
```

This function is shared by both active and archived — the `Active` flag comes from the CSV data itself (`"Yes"` or `"No"`).

---

## 5. Reconcile Lambda — Disappearance resolution

**File:** `pipeline/cmd/lambda/reconcile/main.go` (lines 260-269)

```go
if activeRunID != uuid.Nil {
    for _, archivedRunID := range archivedRunIDs {
        resolved, err := h.Store.ResolveExpectedDisappearances(ctx, activeRunID, archivedRunID, snapshotDate)
        if err != nil {
            h.Logger.Error("failed to resolve expected disappearances", "error", err, "archived_run_id", archivedRunID)
        } else if resolved > 0 {
            h.Logger.Info("expected disappearances resolved (archived)", "count", resolved, "archived_run_id", archivedRunID)
        }
    }

    deactivated, err := h.Store.MarkDisappearedInactive(ctx, activeRunID)
    // ...
}
```

---

## 6. Database — `ResolveExpectedDisappearances`

**File:** `pipeline/internal/database/opportunities.go` (lines 263-281)

```go
func (db *DB) ResolveExpectedDisappearances(ctx context.Context, activeRunID, archivedRunID uuid.UUID, snapshotDate time.Time) (int, error) {
    tag, err := db.pool.Exec(ctx, `
        UPDATE pipeline.snap_disappearances d
        SET resolution = 'archived',
            resolution_date = $3,
            resolution_source = 'archived_csv'
        FROM pipeline.snap_csv a
        WHERE a.notice_id = d.notice_id
          AND a.run_id = $2
          AND d.run_id = $1
          AND d.resolution IS NULL
    `, activeRunID, archivedRunID, snapshotDate)
    // ...
}
```

This JOINs `snap_disappearances` against `snap_csv` rows from the archived run. If a disappeared notice_id is found in the archived CSV, its disappearance is resolved as expected.

---

## 7. Database — `MarkDisappearedInactive`

**File:** `pipeline/internal/database/opportunities.go` (lines 247-261)

```go
func (db *DB) MarkDisappearedInactive(ctx context.Context, runID uuid.UUID) (int, error) {
    tag, err := db.pool.Exec(ctx, `
        UPDATE opportunities o
        SET active = false
        FROM pipeline.snap_disappearances d
        WHERE d.notice_id = o.notice_id
          AND d.run_id = $1
          AND d.resolution IS NULL
          AND o.is_latest = true
    `, runID)
    // ...
}
```

Only acts on **unresolved** disappearances (`resolution IS NULL`). Records resolved as "archived" in step 6 are skipped.

---

## 8. Database — `UpsertOpportunities`

**File:** `pipeline/internal/database/opportunities.go` (lines 22-209)

Shared by both active and archived records. Archived records typically hit one of:
- **New notice_id** → INSERT with `active = false` (version 1)
- **Same hash** → metadata-only UPDATE: `SET active = $3` (sets `false` for archived, then `true` if active overlay follows)
- **Different hash** → new version created

The content hash **excludes** the `Active` field (see `reconcile.go:84-91`), so the same notice in both feeds doesn't create a spurious version.

---

## 9. Store interface methods involved

**File:** `pipeline/internal/database/store.go`

```go
type Store interface {
    // Used by ingest-archived:
    BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, rows) (int64, error)

    // Used by reconcile for archived loading:
    GetSnapCSVRawData(ctx, runID) ([]map[string]string, error)

    // Used by reconcile for disappearance resolution:
    ResolveExpectedDisappearances(ctx, activeRunID, archivedRunID, snapshotDate) (int, error)
    MarkDisappearedInactive(ctx, runID) (int, error)

    // Shared (used by both active and archived paths):
    CreateIngestionRun(ctx, jobType, pipelineRunID) (uuid.UUID, error)
    CompleteIngestionRun(ctx, runID, stats) error
    FailIngestionRun(ctx, runID, errMsg, durationMs) error
    InsertDataQualityIssues(ctx, runID, entries)
    UpsertOpportunities(ctx, runID, snapshotDate, opps) (int, error)
}
```

---

## 10. MockStore fields for archived-related methods

**File:** `pipeline/internal/testutil/mock_store.go`

```go
ResolveExpectedDisappearancesFn func(ctx context.Context, activeRunID, archivedRunID uuid.UUID, snapshotDate time.Time) (int, error)
```

---

## 11. Infrastructure references

### Terraform — Lambda definition
**File:** `infra/terraform/pipeline.tf`
```hcl
lambda_functions = ["download-csvs", "ingest-active", "ingest-archived", "reconcile", "generate-alerts", "ingest-api"]

"ingest-archived" = 512  // memory MB
```

### Terraform — Step Functions ARN mapping
**File:** `infra/terraform/pipeline.tf:167`
```hcl
ingest_archived_arn = aws_lambda_function.pipeline["ingest-archived"].arn
```

### S3 lifecycle rule
**File:** `infra/terraform/data-bucket.tf` (lines 53-65)
```hcl
rule {
    id     = "raw-archived-csv-to-ia"
    status = "Enabled"
    filter {
        prefix = "raw/archived-csv/"
    }
    transition {
        days          = 30
        storage_class = "STANDARD_IA"
    }
}
```

### Makefile
**File:** `Makefile`
```makefile
LAMBDA_FUNCTIONS := download-csvs ingest-active ingest-archived reconcile generate-alerts ingest-api
```

---

## 12. E2E test references

### Fixtures
**File:** `pipeline/internal/e2e/fixtures_test.go` (lines 167-193)

```go
// Archived CSV: static across all 3 runs
var archivedCSV = csvHeaders + "\n" +
    strings.Join([]string{
        // WILL-DISAPPEAR: resolves its disappearance as "archived"
        row(map[int]string{
            0: "WILL-DISAPPEAR", 1: "SOL-DISAP", 2: "Disappearing Record", 3: "Solicitation", 4: "Solicitation",
            5: "01/01/2026", 7: "03/01/2026", 8: "auto", 11: "541511", 13: "No",
            14: "DoD", 17: "097",
        }),
        // ARCHIVED-ONLY: only in archived, never in active
        row(map[int]string{
            0: "ARCHIVED-ONLY", 1: "SOL-ARCH", 2: "Archived Only Opportunity", 3: "Solicitation", 4: "Solicitation",
            5: "11/01/2025", 7: "01/15/2026", 8: "auto", 11: "541511", 13: "No",
            14: "DoD", 17: "097",
        }),
        // HAPPY-001: overlap with active (upsert idempotency)
        row(map[int]string{...}),
    }, "\n")
```

### E2E helpers — `simulateReconcile`
**File:** `pipeline/internal/e2e/helpers_test.go` (lines 78-105)

```go
func simulateReconcile(ctx context.Context, store database.Store, activeRunID uuid.UUID, archivedRunIDs []uuid.UUID, snapshotDate time.Time) error {
    // Archived first so active CSV gets the last word on the active flag
    for _, runID := range archivedRunIDs {
        if err := upsertRun(ctx, store, runID, snapshotDate); err != nil {
            return err
        }
    }
    // ...active upsert...
    // ...ResolveExpectedDisappearances...
    // ...MarkDisappearedInactive...
}
```

### E2E pipeline test — archived-specific specs
**File:** `pipeline/internal/e2e/pipeline_test.go`

- Line 45: `It("ingests archived CSV with 3 records", ...)`
- Line 62-64: Verifies archived `snap_csv` row count
- Line 92: `// Total: 8 active + 3 archived, but HAPPY-001 and WILL-DISAPPEAR overlap → 9 unique`
- Line 106: `// Archived CSV runs second and overwrites with active=false`
- Line 142: `It("ingests archived CSV run 2", ...)`
- Line 177: `It("resolves WILL-DISAPPEAR disappearance as archived", ...)`
- Line 259: `It("ingests archived CSV run 3", ...)`

### E2E API enrichment test
**File:** `pipeline/internal/e2e/api_enrichment_test.go` (lines 28-84)

```go
var apiEnrichmentArchivedCSV = csvHeaders + "\n" +
    row(map[int]string{
        0: "API-ENR-ARCHIVED", 2: "Archived Placeholder", 3: "Solicitation", 13: "No",
    })

// Used in test:
archivedRunID, err := simulateIngest(ctx, db, apiEnrichmentArchivedCSV, "ingest-archived", csvSnapshot)
err = simulateReconcile(ctx, db, activeRunID, []uuid.UUID{archivedRunID}, csvSnapshot)
```

---

## 13. Database schema (from E2E inline SQL)

**File:** `pipeline/internal/e2e/e2e_suite_test.go`

### `pipeline.snap_csv` (lines ~110-130)
```sql
CREATE TABLE pipeline.snap_csv (
    -- ...
    archive_date    TEXT,
    archive_type    TEXT,
    -- ...
    active          BOOLEAN,
    -- ...
    run_id          UUID,
    -- UNIQUE (notice_id, run_id)
);
```

### `pipeline.snap_disappearances` (lines ~145-165)
```sql
CREATE TABLE pipeline.snap_disappearances (
    -- ...
    last_archive_type TEXT,
    last_archive_date TEXT,
    resolution        TEXT,          -- NULL, 'archived', 'glitch'
    resolution_date   TIMESTAMPTZ,
    resolution_source TEXT,          -- 'archived_csv', 'active_csv_reappeared'
    -- ...
);
```

### `public.opportunities` (lines ~185-220)
```sql
CREATE TABLE opportunities (
    -- ...
    archive_date    DATE,
    archive_type    TEXT,
    active          BOOLEAN,
    -- ...
);
```

---

## Summary: What to move

To extract archived CSV handling into its own pipeline, the following need to change:

| Component | What to do |
|-----------|-----------|
| `ingest-archived` Lambda | Move entirely — this is 100% archived-specific |
| `bulkcsv/sources.go` | Extract `SourceTypeArchived` entries |
| Step Functions ASL | Remove `IngestArchived` branch from Map iterator |
| Reconcile Lambda | Remove `archivedRunIDs` loading loop and `ResolveExpectedDisappearances` calls |
| `ResolveExpectedDisappearances` DB method | Move — only used for archived resolution |
| `MarkDisappearedInactive` | Keep but simplify — without archived resolution, all disappearances are unresolved |
| Terraform `pipeline.tf` | Remove `ingest-archived` from Lambda list |
| `data-bucket.tf` | Move S3 lifecycle rule |
| Makefile | Remove `ingest-archived` from `LAMBDA_FUNCTIONS` |
| E2E tests | Update to not include archived runs (or create separate archived E2E suite) |
