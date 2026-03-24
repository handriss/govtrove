.PHONY: help dev-up dev-up-d dev-down dev-clean dev dev-stop \
	install-migrate migrate-up migrate-down migrate-neon migrate-create \
	api-run api-run-d api-run-neon api-stop api-build api-docker-build \
	mcp-install mcp-dev mcp-docker-build \
	frontend-install frontend-dev frontend-dev-d frontend-stop frontend-build \
	test test-e2e test-api-search lambda-build \
	ecr-login deploy-frontend minify-landing deploy-landing deploy-api deploy-mcp deploy-pipeline deploy-all \
	run-pipeline run-pipeline-force pipeline-status pipeline-dlq-status \
	logs-pipeline logs-api status \
	tf-init tf-plan tf-apply tf-output tf-destroy tf-fmt tf-validate \
	n8n-up n8n-down n8n-logs n8n-pull n8n-push n8n-watch \
	clean

# ============================================================================
# Configuration
# ============================================================================

# Auto-load .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

AWS_PROFILE := govtrove
AWS_REGION := us-east-1
LOCAL_DB_URL := postgres://govtrove:localdev@localhost:5432/govtrove?sslmode=disable

# ============================================================================
# Help
# ============================================================================

help:
	@echo "GovTrove Development Commands"
	@echo ""
	@echo "Local Development:"
	@echo "  make dev              - Start everything (db + api + frontend) in background"
	@echo "  make dev-stop         - Stop all local dev services"
	@echo "  make dev-up           - Start local PostgreSQL + Adminer (foreground)"
	@echo "  make dev-up-d         - Start local PostgreSQL + Adminer (detached)"
	@echo "  make dev-down         - Stop local database services"
	@echo ""
	@echo "n8n Automations:"
	@echo "  make n8n-up           - Start n8n (localhost:5678)"
	@echo "  make n8n-down         - Stop n8n"
	@echo "  make n8n-logs         - Tail n8n logs"
	@echo "  make n8n-pull         - Pull workflows from n8n to n8n/workflows/"
	@echo "  make n8n-push         - Push workflows from n8n/workflows/ into n8n"
	@echo "  make n8n-watch        - Auto-pull workflows every 30s"
	@echo ""
	@echo "API Service:"
	@echo "  make api-run          - Run API server locally (localhost:3000)"
	@echo "  make api-run-d        - Run API server in background"
	@echo "  make api-run-neon     - Run API locally against Neon DB"
	@echo "  make api-stop         - Stop background API server"
	@echo "  make api-build        - Build API binary"
	@echo "  make api-docker-build - Build API Docker image"
	@echo ""
	@echo "Frontend:"
	@echo "  make frontend-dev     - Run frontend dev server (localhost:5173)"
	@echo "  make frontend-dev-d   - Run frontend dev server in background"
	@echo "  make frontend-stop    - Stop background frontend server"
	@echo "  make frontend-build   - Build frontend for production"
	@echo ""
	@echo "Pipeline (production):"
	@echo "  make run-pipeline                 - Manually trigger Step Functions pipeline"
	@echo "  make run-pipeline-force           - Clear CSV cache and trigger pipeline (full re-download)"
	@echo "  make pipeline-status              - Show recent pipeline executions"
	@echo "  make pipeline-dlq-status          - Check DLQ depth"
	@echo "  make logs-pipeline SVC=<name>     - Tail logs for pipeline Lambda"
	@echo "  (Lambda functions: download-csvs, ingest-active, reconcile, generate-alerts, ingest-api)"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up       - Run migrations (local DB)"
	@echo "  make migrate-down     - Rollback migrations (local DB)"
	@echo "  make migrate-neon     - Run migrations (Neon DB)"
	@echo ""
	@echo "Build:"
	@echo "  make lambda-build       - Build all 4 pipeline Lambda zips"
	@echo "  make test               - Run tests"
	@echo "  make test-e2e           - Run pipeline E2E tests (requires Docker)"
	@echo ""
	@echo "Deploy:"
	@echo "  make ecr-login                  - Login to AWS ECR"
	@echo "  make deploy-landing             - Deploy landing page to S3/CloudFront"
	@echo "  make deploy-frontend            - Build & deploy frontend app to S3/CloudFront"
	@echo "  make deploy-api                 - Build & deploy API to App Runner"
	@echo "  make deploy-pipeline            - Build & deploy pipeline Lambda functions"
	@echo "  make deploy-all                 - Deploy everything"
	@echo ""
	@echo "Operations:"
	@echo "  make logs-pipeline SVC=x - Tail logs for a pipeline Lambda"
	@echo "  make logs-api            - Tail App Runner logs for API"
	@echo "  make status              - Show status of deployed services"
	@echo ""
	@echo "Terraform:"
	@echo "  make tf-init          - Initialize Terraform"
	@echo "  make tf-plan          - Plan Terraform changes"
	@echo "  make tf-apply         - Apply Terraform changes (with confirmation)"
	@echo "  make tf-output        - Show Terraform outputs"
	@echo "  make tf-destroy       - Destroy infrastructure (with confirmation)"
	@echo "  make tf-fmt           - Format Terraform files"
	@echo "  make tf-validate      - Validate Terraform syntax"
	@echo ""

# ============================================================================
# Local Development
# ============================================================================

dev-up:
	docker compose -f infra/docker-compose.yml up

dev-up-d:
	docker compose -f infra/docker-compose.yml up -d

dev-down:
	docker compose -f infra/docker-compose.yml down

dev-clean:
	docker compose -f infra/docker-compose.yml down -v

dev: dev-up-d migrate-up api-run-d frontend-dev-d
	@echo ""
	@echo "Local dev environment started:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Adminer:    http://localhost:8080"
	@echo "  - API:        http://localhost:3000"
	@echo "  - Frontend:   http://localhost:5173"
	@echo "  - n8n:        http://localhost:5678"
	@echo ""
	@echo "Run 'make dev-stop' to stop all services"

dev-stop: frontend-stop api-stop dev-down
	@echo "All local dev services stopped"

# ============================================================================
# n8n Automations
# ============================================================================

N8N_COMPOSE := docker compose -f infra/docker-compose.yml

n8n-up:
	$(N8N_COMPOSE) up -d n8n

n8n-down:
	$(N8N_COMPOSE) stop n8n

n8n-logs:
	$(N8N_COMPOSE) logs -f n8n

n8n-pull:
	@echo "Pulling n8n workflows..."
	@$(N8N_COMPOSE) exec -T n8n \
		sh -c 'rm -rf /home/node/.n8n/exports && mkdir -p /home/node/.n8n/exports && n8n export:workflow --all --output=/home/node/.n8n/exports/ --separate'
	@mkdir -p /tmp/n8n-pull
	@rm -f /tmp/n8n-pull/*.json
	@$(N8N_COMPOSE) cp n8n:/home/node/.n8n/exports/. /tmp/n8n-pull/
	@changed=0; \
	for f in /tmp/n8n-pull/*.json; do \
		[ -f "$$f" ] || continue; \
		name=$$(basename "$$f"); \
		if [ ! -f "n8n/workflows/$$name" ] || ! diff -q "$$f" "n8n/workflows/$$name" >/dev/null 2>&1; then \
			cp "$$f" "n8n/workflows/$$name"; \
			changed=$$((changed + 1)); \
		fi; \
	done; \
	for f in n8n/workflows/*.json; do \
		[ -f "$$f" ] || continue; \
		name=$$(basename "$$f"); \
		if [ ! -f "/tmp/n8n-pull/$$name" ]; then \
			rm "$$f"; \
			changed=$$((changed + 1)); \
		fi; \
	done; \
	rm -rf /tmp/n8n-pull; \
	if [ $$changed -gt 0 ]; then \
		echo "Updated $$changed workflow file(s)"; \
	else \
		echo "No changes"; \
	fi

n8n-push:
	@echo "Pushing workflows to n8n..."
	@if ls n8n/workflows/*.json 1>/dev/null 2>&1; then \
		$(N8N_COMPOSE) cp n8n/workflows/. n8n:/home/node/.n8n/imports/ && \
		$(N8N_COMPOSE) exec -T n8n \
			sh -c 'for f in /home/node/.n8n/imports/*.json; do n8n import:workflow --input="$$f"; done && rm -rf /home/node/.n8n/imports' && \
		echo "Pushed workflows to n8n"; \
	else \
		echo "No workflow files found in n8n/workflows/"; \
	fi

n8n-watch:
	@echo "Watching n8n for workflow changes (every 30s)..."
	@echo "Press Ctrl+C to stop"
	@while true; do \
		$(MAKE) -s n8n-pull 2>/dev/null || echo "n8n not running — retrying..."; \
		sleep 30; \
	done

# ============================================================================
# Migrations
# ============================================================================

install-migrate:
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up: install-migrate
	migrate -path ./infra/migrations -database "$(LOCAL_DB_URL)" up

migrate-down: install-migrate
	migrate -path ./infra/migrations -database "$(LOCAL_DB_URL)" down

migrate-neon: install-migrate
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	migrate -path ./infra/migrations -database "$(NEON_DATABASE_URL)" up

migrate-create: install-migrate
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir ./infra/migrations -seq $$name

# ============================================================================
# API Service
# ============================================================================

api-run:
	cd api && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	LOG_LEVEL=debug \
	go run ./cmd/api

api-run-d:
	@mkdir -p .logs
	@echo "Starting API server in background..."
	@cd api && DATABASE_URL="$(LOCAL_DB_URL)" LOG_LEVEL=debug \
		nohup go run ./cmd/api > ../.logs/api.log 2>&1 & echo $$! > ../.logs/api.pid
	@sleep 3
	@echo "API server started (PID: $$(cat .logs/api.pid))"
	@echo "Logs: .logs/api.log"

api-run-neon:
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	cd api && DATABASE_URL="$(NEON_DATABASE_URL)" LOG_LEVEL=debug go run ./cmd/api

api-stop:
	@if [ -f .logs/api.pid ]; then \
		PID=$$(cat .logs/api.pid); \
		kill $$PID 2>/dev/null || true; \
		pkill -P $$PID 2>/dev/null || true; \
		rm -f .logs/api.pid; \
		echo "API server stopped"; \
	else \
		echo "No API server PID file found"; \
	fi

api-build:
	cd api && go build -o ../bin/api ./cmd/api

api-docker-build:
	docker build --platform linux/amd64 -t govtrove-api:latest ./api

# ============================================================================
# MCP Server
# ============================================================================

mcp-install:
	cd mcp && npm install

mcp-dev:
	cd mcp && npm run dev

mcp-docker-build:
	docker build --platform linux/amd64 -t govtrove-mcp:latest ./mcp

# ============================================================================
# Frontend
# ============================================================================

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-dev-d:
	@mkdir -p .logs
	@echo "Starting frontend dev server in background..."
	@(cd frontend && nohup npm run dev > ../.logs/frontend.log 2>&1) & echo $$! > .logs/frontend.pid
	@sleep 2
	@echo "Frontend dev server started (PID: $$(cat .logs/frontend.pid))"
	@echo "Logs: .logs/frontend.log"

frontend-stop:
	@if [ -f .logs/frontend.pid ]; then \
		PID=$$(cat .logs/frontend.pid); \
		kill $$PID 2>/dev/null || true; \
		pkill -P $$PID 2>/dev/null || true; \
		rm -f .logs/frontend.pid; \
		echo "Frontend server stopped"; \
	else \
		echo "No frontend server PID file found"; \
	fi

frontend-build:
	cd frontend && npm run build

# ============================================================================
# Build
# ============================================================================

LAMBDA_FUNCTIONS := download-csvs ingest-active reconcile generate-alerts ingest-api seo-trends seo-pages

seo-pages:
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	cd pipeline && NEON_DATABASE_URL="$(NEON_DATABASE_URL)" go run ./cmd/seo-pages/ -out ../landing/contracts
	@echo "SEO pages generated in landing/contracts/"

test:
	cd pipeline && go test -v ./...
	cd api && go test -v ./...

test-e2e:
	@mkdir -p .reports
	cd pipeline && go test -v ./internal/e2e/ -timeout 120s -ginkgo.json-report=../../../.reports/e2e-report.json
	@echo "Report saved to .reports/e2e-report.json"

test-api-search:
	cd api && go test -v -count=1 ./internal/searchtest/ -timeout 120s

lambda-build:
	@for svc in $(LAMBDA_FUNCTIONS); do \
		echo "Building Lambda: $$svc..."; \
		mkdir -p bin/lambda/$$svc; \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -C pipeline -o ../bin/lambda/$$svc/bootstrap ./cmd/lambda/$$svc; \
	done
	@echo "All Lambda functions built"

# ============================================================================
# Deploy
# ============================================================================

ecr-login:
	@AWS_ACCOUNT_ID=$$(aws sts get-caller-identity --query Account --output text --profile $(AWS_PROFILE)) && \
	aws ecr get-login-password --region $(AWS_REGION) --profile $(AWS_PROFILE) | \
		docker login --username AWS --password-stdin $$AWS_ACCOUNT_ID.dkr.ecr.$(AWS_REGION).amazonaws.com

deploy-frontend: frontend-build
	@echo "Deploying frontend to S3/CloudFront..."
	@API_URL=$$(cd infra/terraform && terraform output -raw api_url) && \
	WORKOS_CLIENT_ID=$$(cd infra/terraform && terraform output -raw workos_client_id 2>/dev/null || echo "") && \
	SENTRY_DSN=$$(cd infra/terraform && terraform output -raw sentry_frontend_dsn 2>/dev/null || echo "") && \
	TAWK_ID=$$(cd infra/terraform && terraform output -raw tawk_property_id 2>/dev/null || echo "") && \
	echo "VITE_API_URL=$$API_URL/api" > .env.production && \
	if [ -n "$$WORKOS_CLIENT_ID" ]; then echo "VITE_WORKOS_CLIENT_ID=$$WORKOS_CLIENT_ID" >> .env.production; fi && \
	if [ -n "$$SENTRY_DSN" ]; then echo "VITE_SENTRY_DSN=$$SENTRY_DSN" >> .env.production; fi && \
	if [ -n "$$TAWK_ID" ]; then echo "VITE_TAWK_PROPERTY_ID=$$TAWK_ID" >> .env.production; fi && \
	cd frontend && npm run build && \
	BUCKET=$$(cd ../infra/terraform && terraform output -raw frontend_bucket_name) && \
	DIST_ID=$$(cd ../infra/terraform && terraform output -raw cloudfront_distribution_id) && \
	echo "Syncing hashed assets (immutable cache)..." && \
	aws s3 sync dist s3://$$BUCKET --delete \
		--exclude "*.html" --exclude "robots.txt" --exclude "sitemap.xml" --exclude "favicon.svg" \
		--cache-control "public, max-age=31536000, immutable" \
		--profile $(AWS_PROFILE) && \
	echo "Syncing HTML + metadata (no-cache)..." && \
	aws s3 sync dist s3://$$BUCKET \
		--exclude "*" --include "*.html" --include "robots.txt" --include "sitemap.xml" --include "favicon.svg" \
		--cache-control "no-cache" \
		--profile $(AWS_PROFILE) && \
	echo "Invalidating CloudFront cache..." && \
	aws cloudfront create-invalidation --distribution-id $$DIST_ID --paths "/*" --profile $(AWS_PROFILE) && \
	rm -f ../.env.production && \
	echo "Frontend deployed successfully!"

deploy-api: api-docker-build ecr-login
	@echo "Deploying API to App Runner..."
	@ECR_URL=$$(cd infra/terraform && terraform output -raw ecr_api_repository_url) && \
	ARN=$$(cd infra/terraform && terraform output -raw apprunner_service_arn) && \
	docker tag govtrove-api:latest $$ECR_URL:latest && \
	docker push $$ECR_URL:latest && \
	echo "Triggering App Runner deployment..." && \
	aws apprunner start-deployment --service-arn $$ARN --profile $(AWS_PROFILE) --region $(AWS_REGION) && \
	echo "API deployment triggered! Check status with 'make status'"

deploy-mcp: mcp-docker-build ecr-login
	@echo "Deploying MCP server to App Runner..."
	@ECR_URL=$$(cd infra/terraform && terraform output -raw ecr_mcp_repository_url) && \
	ARN=$$(cd infra/terraform && terraform output -raw mcp_service_arn) && \
	docker tag govtrove-mcp:latest $$ECR_URL:latest && \
	docker push $$ECR_URL:latest && \
	echo "Triggering App Runner deployment..." && \
	aws apprunner start-deployment --service-arn $$ARN --profile $(AWS_PROFILE) --region $(AWS_REGION) && \
	echo "MCP deployment triggered! Check status with 'make status'"

deploy-pipeline: lambda-build
	@echo "Deploying pipeline Lambda functions..."
	@for svc in $(LAMBDA_FUNCTIONS); do \
		echo "Updating $$svc..."; \
		cd bin/lambda/$$svc && zip -j ../$$svc.zip bootstrap && cd ../../..; \
		aws lambda update-function-code \
			--function-name govtrove-$$svc \
			--zip-file fileb://bin/lambda/$$svc.zip \
			--profile $(AWS_PROFILE) --region $(AWS_REGION) > /dev/null; \
	done
	@echo "All pipeline Lambda functions deployed!"

minify-landing:
	@echo "Minifying landing page..."
	@rm -rf landing-dist
	@cp -r landing landing-dist
	@find landing-dist -name '*.html' -exec npx --yes html-minifier-terser \
		--collapse-whitespace --remove-comments --remove-redundant-attributes \
		--minify-css true --minify-js true \
		-o {} {} \;
	@find landing-dist -name '*.js' ! -name '*.min.js' -exec npx --yes terser {} -o {} --compress --mangle \;
	@find landing-dist -name '*.css' -exec npx --yes csso-cli {} -o {} \;
	@echo "Minification complete → landing-dist/"

deploy-landing: minify-landing
	@echo "Deploying landing page to S3/CloudFront..."
	@BUCKET=$$(cd infra/terraform && terraform output -raw landing_bucket_name) && \
	DIST_ID=$$(cd infra/terraform && terraform output -raw landing_distribution_id) && \
	aws s3 sync landing-dist s3://$$BUCKET --delete --exclude "contracts/*" --exclude "sitemap.xml" --profile $(AWS_PROFILE) && \
	echo "Invalidating CloudFront cache..." && \
	aws cloudfront create-invalidation --distribution-id $$DIST_ID --paths "/*" --profile $(AWS_PROFILE) && \
	rm -rf landing-dist && \
	echo "Landing page deployed successfully!"

deploy-all: deploy-api deploy-pipeline deploy-frontend deploy-landing
	@echo ""
	@echo "All services deployed!"

# ============================================================================
# Pipeline Operations
# ============================================================================

run-pipeline:
	@echo "Starting Step Functions pipeline execution..."
	@ARN=$$(cd infra/terraform && terraform output -raw pipeline_state_machine_arn) && \
	aws stepfunctions start-execution \
		--state-machine-arn $$ARN \
		--profile $(AWS_PROFILE) --region $(AWS_REGION) && \
	echo "Pipeline execution started! Check status with 'make pipeline-status'"

run-pipeline-force:
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	@echo "Clearing CSV cache to force re-download..."
	@psql "$(NEON_DATABASE_URL)" -c \
		"DELETE FROM pipeline.bulk_csv_log WHERE id = (SELECT id FROM pipeline.bulk_csv_log WHERE source = 'active' ORDER BY checked_at DESC LIMIT 1);" && \
	echo "Cache cleared."
	@echo "Starting Step Functions pipeline execution..."
	@ARN=$$(cd infra/terraform && terraform output -raw pipeline_state_machine_arn) && \
	aws stepfunctions start-execution \
		--state-machine-arn $$ARN \
		--profile $(AWS_PROFILE) --region $(AWS_REGION) && \
	echo "Pipeline execution started! Check status with 'make pipeline-status'"

pipeline-status:
	@echo "=== Recent Pipeline Executions ==="
	@ARN=$$(cd infra/terraform && terraform output -raw pipeline_state_machine_arn) && \
	aws stepfunctions list-executions \
		--state-machine-arn $$ARN \
		--max-results 5 \
		--query 'executions[].{Name:name,Status:status,Start:startDate,Stop:stopDate}' \
		--output table \
		--profile $(AWS_PROFILE) --region $(AWS_REGION)

pipeline-dlq-status:
	@echo "=== Pipeline DLQ Depth ==="
	@DLQ_URL=$$(cd infra/terraform && terraform output -raw pipeline_dlq_url) && \
	COUNT=$$(aws sqs get-queue-attributes --queue-url "$$DLQ_URL" \
		--attribute-names ApproximateNumberOfMessages \
		--query 'Attributes.ApproximateNumberOfMessages' --output text \
		--profile $(AWS_PROFILE) --region $(AWS_REGION)) && \
	echo "  pipeline-dlq: $$COUNT messages"

# ============================================================================
# Operations
# ============================================================================

logs-pipeline:
	@if [ -z "$(SVC)" ]; then echo "Usage: make logs-pipeline SVC=<name>"; echo "Functions: download-csvs, ingest-active, reconcile, generate-alerts, ingest-api"; exit 1; fi
	aws logs tail /aws/lambda/govtrove-$(SVC) --follow --profile $(AWS_PROFILE) --region $(AWS_REGION)

logs-api:
	@SERVICE_ARN=$$(cd infra/terraform && terraform output -raw apprunner_service_arn) && \
	SERVICE_ID=$$(echo $$SERVICE_ARN | rev | cut -d'/' -f1 | rev) && \
	aws logs tail /aws/apprunner/govtrove-api/$$SERVICE_ID/application --follow --profile $(AWS_PROFILE) --region $(AWS_REGION)

status:
	@echo "=== App Runner API ===" && \
	ARN=$$(cd infra/terraform && terraform output -raw apprunner_service_arn 2>/dev/null) && \
	if [ -n "$$ARN" ]; then \
		aws apprunner describe-service --service-arn $$ARN \
			--query 'Service.{Status:Status,URL:ServiceUrl,Updated:UpdatedAt}' \
			--output table --profile $(AWS_PROFILE) --region $(AWS_REGION); \
	else \
		echo "App Runner service not found (run tf-apply first)"; \
	fi
	@echo ""
	@echo "=== CloudFront Frontend ===" && \
	DIST_ID=$$(cd infra/terraform && terraform output -raw cloudfront_distribution_id 2>/dev/null) && \
	if [ -n "$$DIST_ID" ]; then \
		aws cloudfront get-distribution --id $$DIST_ID \
			--query 'Distribution.{Status:Status,DomainName:DomainName}' \
			--output table --profile $(AWS_PROFILE); \
	else \
		echo "CloudFront distribution not found (run tf-apply first)"; \
	fi
	@echo ""
	@echo "=== Recent Pipeline Executions ===" && \
	SFN_ARN=$$(cd infra/terraform && terraform output -raw pipeline_state_machine_arn 2>/dev/null) && \
	if [ -n "$$SFN_ARN" ]; then \
		aws stepfunctions list-executions \
			--state-machine-arn $$SFN_ARN \
			--max-results 5 \
			--query 'executions[].{Name:name,Status:status,Start:startDate}' \
			--output table --profile $(AWS_PROFILE) --region $(AWS_REGION); \
	else \
		echo "Step Functions state machine not found (run tf-apply first)"; \
	fi

# ============================================================================
# Terraform
# ============================================================================

tf-init:
	cd infra/terraform && terraform init

tf-plan:
	cd infra/terraform && terraform plan

tf-apply:
	cd infra/terraform && terraform apply

tf-output:
	cd infra/terraform && terraform output

tf-destroy:
	cd infra/terraform && terraform destroy

tf-fmt:
	cd infra/terraform && terraform fmt

tf-validate:
	cd infra/terraform && terraform validate

# ============================================================================
# Cleanup
# ============================================================================

clean:
	rm -rf bin/
	rm -rf .logs/
	rm -rf frontend/node_modules frontend/dist
	docker compose -f infra/docker-compose.yml down -v