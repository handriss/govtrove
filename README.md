# GovTrove

Federal contract opportunity search platform. Aggregates data from SAM.gov's official public data services and provides a fast, searchable interface for government contractors.

## Architecture

```
                    govtrove.com          app.govtrove.com         api.govtrove.com
                         |                       |                       |
                    CloudFront              CloudFront              Cloudflare
                    (Landing)               (Frontend)              (Proxy)
                         |                       |                       |
                    S3 Bucket               S3 Bucket              App Runner
                                                                     (API)
                                                                       |
+------------------+                                            +-----------+
|  EventBridge     |  every 15 min                              |  Neon DB  |
|  (Schedule)      |----------+                                 | (Postgres)|
+------------------+          |                                 +-----------+
                              v                                       ^
                    +------------------+                               |
                    |  Step Functions  |-------------------------------+
                    |  (Pipeline)      |
                    +------------------+
                              |
               +--------------+--------------+
               |              |              |
          download-csvs  ingest-active  reconcile
                         ingest-archived
                         ingest-api
                         generate-alerts
```

**Components:**
- **Landing page** — Static marketing site (`landing/`), served via CloudFront/S3 at govtrove.com
- **Frontend** — React SPA (`frontend/`), served via CloudFront/S3 at app.govtrove.com
- **API** — Go service (`api/`), running on AWS App Runner at api.govtrove.com (proxied through Cloudflare)
- **Pipeline** — Go Lambda functions (`pipeline/`), orchestrated by Step Functions on a 15-minute schedule. Downloads SAM.gov CSV bulk exports and API data, ingests into the database, and reconciles changes.
- **Database** — Neon serverless PostgreSQL

## Prerequisites

- Go 1.24+
- Node.js 20+
- Docker (for local PostgreSQL and Lambda builds)
- AWS CLI v2 (configured with `govtrove` profile)
- Terraform 1.0+

```bash
# Configure AWS profile (one-time setup)
aws configure --profile govtrove
```

## Local Development

### Quick Start

```bash
# Start everything (database + API + frontend)
make dev

# Stop everything
make dev-stop
```

This starts:
- PostgreSQL at `localhost:5432`
- Adminer (DB UI) at `http://localhost:8080`
- API at `http://localhost:3000`
- Frontend at `http://localhost:5173`

### Running Services Individually

```bash
# Database
make dev-up-d        # Start PostgreSQL + Adminer (background)
make dev-down        # Stop
make dev-clean       # Stop and delete data volumes

# API (requires database running)
make api-run         # Foreground
make api-run-d       # Background (logs at .logs/api.log)
make api-stop        # Stop background process

# API against Neon (production data, local code)
make api-run-neon    # Requires NEON_DATABASE_URL in .env

# Frontend
make frontend-dev    # Foreground (localhost:5173)
make frontend-dev-d  # Background (logs at .logs/frontend.log)
make frontend-stop   # Stop background process
```

### Database Migrations

```bash
make migrate-up       # Run migrations (local DB)
make migrate-down     # Rollback migrations (local DB)
make migrate-create   # Create a new migration (prompts for name)
make migrate-neon     # Run migrations on Neon (requires NEON_DATABASE_URL)
```

### Testing

```bash
make test             # Run all unit tests (pipeline + api)
make test-e2e         # Run pipeline E2E tests (requires Docker for testcontainers)
```

## Deployment

All deploy commands use the `govtrove` AWS profile and read configuration from Terraform outputs.

### Deploy Everything

```bash
make deploy-all
```

### Deploy Individually

```bash
make deploy-frontend    # Build React app → S3 → CloudFront invalidation
make deploy-landing     # Sync landing/ → S3 → CloudFront invalidation
make deploy-api         # Docker build → ECR push → App Runner deployment
make deploy-pipeline    # Cross-compile Lambda zips → update-function-code
```

**Frontend deployment** automatically pulls the API URL, WorkOS client ID, and Sentry DSN from Terraform outputs and writes a temporary `.env.production` for the Vite build.

**Landing page deployment** syncs the entire `landing/` directory to S3 with `--delete`.

**API deployment** builds a linux/amd64 Docker image, pushes to ECR, and triggers an App Runner deployment.

**Pipeline deployment** cross-compiles all 6 Lambda functions (download-csvs, ingest-active, ingest-archived, ingest-api, reconcile, generate-alerts) as `provided.al2023` binaries, zips them, and updates each function via `aws lambda update-function-code`.

## Pipeline Operations

```bash
make run-pipeline           # Manually trigger the Step Functions pipeline
make pipeline-status        # Show last 5 pipeline executions
make pipeline-dlq-status    # Check dead letter queue depth
```

## Operations

```bash
make logs-api                       # Tail App Runner logs
make logs-pipeline SVC=reconcile    # Tail a specific Lambda's logs
make status                         # Show status of all deployed services
```

Available Lambda names for `SVC`: `download-csvs`, `ingest-active`, `ingest-archived`, `ingest-api`, `reconcile`, `generate-alerts`

## Infrastructure

All infrastructure is managed with Terraform in `infra/terraform/`.

```bash
make tf-init       # Initialize (first time or after provider changes)
make tf-plan       # Preview changes
make tf-apply      # Apply changes (with confirmation)
make tf-output     # Show all outputs (URLs, ARNs, bucket names, etc.)
make tf-fmt        # Format .tf files
make tf-validate   # Validate syntax
```

## Environment Variables

**Local development** uses Docker Compose defaults — no `.env` file needed for basic dev.

**For Neon/production database access**, create a `.env` file in the repo root (see `.env.example`):
```bash
NEON_DATABASE_URL=postgres://user:password@host.neon.tech/govtrove?sslmode=require
SAM_API_KEY=your-sam-api-key
```

**Production secrets** (DATABASE_URL, SAM_API_KEY, WorkOS keys) are managed through AWS Secrets Manager and Terraform — never committed to the repo.

## Project Structure

```
govtrove/
├── api/                  # Go API service
│   ├── cmd/api/          # Entrypoint
│   └── internal/         # Handlers, repository, middleware
├── frontend/             # React + Vite frontend
│   └── src/
├── pipeline/             # Go pipeline Lambda functions
│   ├── cmd/lambda/       # Lambda entrypoints (6 functions)
│   └── internal/         # SAM.gov client, CSV parsing, DB ops, reconciler
├── landing/              # Static landing page + blog
│   └── blog/             # Blog post HTML files
├── infra/                # Infrastructure
│   ├── terraform/        # All AWS resources
│   ├── migrations/       # SQL migrations (golang-migrate)
│   ├── cf-functions/     # CloudFront functions (OG redirect)
│   └── docker-compose.yml
├── docs/                 # Documentation
├── Makefile              # All commands
└── .env.example          # Environment variable template
```
