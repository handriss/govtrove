# GovTrove

GovTrove is a federal contracting opportunity search platform that aggregates data from SAM.gov and provides a fast, searchable interface for government contractors.

## Architecture

```
                                    +------------------+
                                    |    CloudFront    |
                                    |    (Frontend)    |
                                    +--------+---------+
                                             |
                                             v
+------------------+              +------------------+
|   SAM.gov API    |              |    App Runner    |
|                  |              |      (API)       |
+--------+---------+              +--------+---------+
         |                                 |
         v                                 v
+------------------+              +------------------+
|   ECS Fargate    |------------->|     Neon DB      |
|   (Ingestion)    |              |   (PostgreSQL)   |
+------------------+              +------------------+
```

**Components:**
- **Frontend** - React SPA served via CloudFront/S3
- **API** - Go service on AWS App Runner
- **Ingestion** - Scheduled Go service on ECS Fargate that pulls data from SAM.gov
- **Database** - Neon serverless PostgreSQL

## Prerequisites

- Go 1.21+
- Node.js 20+
- Docker
- AWS CLI v2
- Terraform 1.0+
- jq (for some operations commands)

**AWS Setup:**
```bash
# Configure AWS profile
aws configure --profile govtrove

# Verify access
aws sts get-caller-identity --profile govtrove
```

## Local Development

### Quick Start

```bash
# Start everything (database, API, frontend)
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
# Database only
make dev-up-d        # Start in background
make dev-down        # Stop

# API only (requires database)
make api-run         # Foreground
make api-run-d       # Background
make api-stop        # Stop background

# Frontend only
make frontend-dev    # Foreground
make frontend-dev-d  # Background
make frontend-stop   # Stop background
```

### Database Migrations

```bash
# Run migrations locally
make migrate-up

# Rollback
make migrate-down

# Create new migration
make migrate-create
# Enter name when prompted

# Run migrations on Neon
NEON_DATABASE_URL="postgres://..." make migrate-neon
```

### Ingestion Service

```bash
# Run against local database
make run

# Run against mock SAM.gov server
make mock-server &    # Start mock server
make run-mock         # Run ingestion

# Run against Neon with real SAM.gov API
NEON_DATABASE_URL="postgres://..." SAM_API_KEY="..." make run-neon
```

### Testing Against Neon

Before deploying, you can test the API locally against the production database:

```bash
NEON_DATABASE_URL="postgres://..." make api-run-neon
```

## Deployment

### Initial Setup

1. Initialize Terraform:
   ```bash
   make tf-init
   ```

2. Review and apply infrastructure:
   ```bash
   make tf-plan
   make tf-apply   # Requires manual confirmation
   ```

3. Run database migrations on Neon:
   ```bash
   NEON_DATABASE_URL="postgres://..." make migrate-neon
   ```

### Deploying Services

**Deploy Everything:**
```bash
make deploy-all
```

**Deploy Individually:**
```bash
# API (builds, pushes to ECR, triggers App Runner)
make deploy-api

# Frontend (builds with prod API URL, syncs to S3, invalidates CloudFront)
make deploy-frontend

# Ingestion (builds, pushes to ECR - runs on schedule)
make deploy-ingestion
```

### Deployment Flow

**API Deployment:**
1. Builds Docker image locally
2. Authenticates with ECR
3. Pushes image to ECR
4. Triggers App Runner deployment

**Frontend Deployment:**
1. Gets App Runner URL from Terraform
2. Builds frontend with production API URL
3. Syncs to S3 bucket
4. Invalidates CloudFront cache

**Ingestion Deployment:**
1. Builds Docker image locally
2. Pushes to ECR
3. ECS scheduled task uses new image on next run

## Operations

### Viewing Logs

```bash
# API logs (App Runner)
make logs-api

# Ingestion logs (CloudWatch)
make logs-ingestion
```

### Service Status

```bash
make status
```

Shows:
- App Runner API status and URL
- CloudFront distribution status
- Recent ingestion task runs

### Manual Ingestion Run

To trigger an ingestion run outside the schedule:

```bash
make run-ingestion-aws
```

## Infrastructure Management

### Terraform Commands

```bash
make tf-init      # Initialize (first time or after provider changes)
make tf-plan      # Preview changes
make tf-apply     # Apply changes (with confirmation)
make tf-output    # Show all outputs (URLs, ARNs, etc.)
make tf-destroy   # Destroy infrastructure (with confirmation)
make tf-fmt       # Format .tf files
make tf-validate  # Validate syntax
```

### Key Terraform Outputs

```bash
# Get specific outputs
cd terraform
terraform output apprunner_service_url      # API URL
terraform output cloudfront_distribution_url # Frontend URL
terraform output ecr_repository_url         # Ingestion ECR
terraform output ecr_api_repository_url     # API ECR
```

## Quick Reference

| Task | Command |
|------|---------|
| Start local dev | `make dev` |
| Stop local dev | `make dev-stop` |
| Run tests | `make test` |
| Deploy API | `make deploy-api` |
| Deploy frontend | `make deploy-frontend` |
| Deploy everything | `make deploy-all` |
| View API logs | `make logs-api` |
| Check status | `make status` |
| Run ingestion manually | `make run-ingestion-aws` |

## Environment Variables

**Local Development:**
- Uses `infra/docker-compose.yml` defaults
- No `.env` file needed for basic dev

**Production (set in Terraform/AWS):**
- `DATABASE_URL` - Neon connection string (via Secrets Manager)
- `SAM_API_KEY` - SAM.gov API key (via Secrets Manager)
- `ALLOWED_ORIGINS` - CORS origins (set automatically from CloudFront URL)

**Local Testing with Neon:**
```bash
export NEON_DATABASE_URL="postgres://user:pass@host/db?sslmode=require"
export SAM_API_KEY="your-sam-api-key"
```

## Project Structure

```
govtrove/
├── api/                  # Go API service
│   ├── cmd/api/          # Main entrypoint
│   └── internal/         # Handlers, repository, config
├── frontend/             # React frontend
│   └── src/
├── pipeline/             # Go pipeline Lambdas (data ingestion)
│   ├── cmd/lambda/       # Lambda entrypoints
│   └── internal/         # SAM.gov client, DB operations
├── landing/              # Static landing page
├── infra/                # Infrastructure
│   ├── terraform/        # Infrastructure as code
│   ├── migrations/       # SQL migrations
│   ├── cf-functions/     # CloudFront functions
│   └── docker-compose.yml
├── docs/                 # Documentation
└── Makefile              # All commands
```
