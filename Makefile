.PHONY: help dev-up dev-up-d dev-down dev-clean dev dev-stop \
	install-migrate migrate-up migrate-down migrate-neon migrate-create \
	api-run api-run-d api-run-neon api-stop api-build api-docker-build \
	frontend-install frontend-dev frontend-dev-d frontend-stop frontend-build \
	run run-neon run-mock run-csv-only mock-server \
	build test ingestion-docker-build \
	ecr-login deploy-frontend deploy-api deploy-ingestion deploy-all \
	logs-ingestion logs-api run-ingestion-aws status \
	tf-init tf-plan tf-apply tf-output tf-destroy tf-fmt tf-validate \
	clean

# ============================================================================
# Configuration
# ============================================================================

AWS_PROFILE := opscout
AWS_REGION := us-east-1
LOCAL_DB_URL := postgres://opscout:localdev@localhost:5432/opscout?sslmode=disable

# ============================================================================
# Help
# ============================================================================

help:
	@echo "OpScout Development Commands"
	@echo ""
	@echo "Local Development:"
	@echo "  make dev              - Start everything (db + api + frontend) in background"
	@echo "  make dev-stop         - Stop all local dev services"
	@echo "  make dev-up           - Start local PostgreSQL + Adminer (foreground)"
	@echo "  make dev-up-d         - Start local PostgreSQL + Adminer (detached)"
	@echo "  make dev-down         - Stop local database services"
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
	@echo "Ingestion Service:"
	@echo "  make run              - Run ingestion service locally (local DB)"
	@echo "  make run-neon         - Run ingestion service locally (Neon DB)"
	@echo "  make run-mock         - Run ingestion against mock SAM.gov server"
	@echo "  make run-csv-only     - Run CSV-only ingestion (skip API)"
	@echo "  make mock-server      - Start mock SAM.gov API server"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up       - Run migrations (local DB)"
	@echo "  make migrate-down     - Rollback migrations (local DB)"
	@echo "  make migrate-neon     - Run migrations (Neon DB)"
	@echo ""
	@echo "Build:"
	@echo "  make build                  - Build ingestion binary"
	@echo "  make ingestion-docker-build - Build ingestion Docker image"
	@echo "  make test                   - Run tests"
	@echo ""
	@echo "Deploy:"
	@echo "  make ecr-login        - Login to AWS ECR"
	@echo "  make deploy-frontend  - Build & deploy frontend to S3/CloudFront"
	@echo "  make deploy-api       - Build & deploy API to App Runner"
	@echo "  make deploy-ingestion - Build & push ingestion image to ECR"
	@echo "  make deploy-all       - Deploy everything"
	@echo ""
	@echo "Operations:"
	@echo "  make logs-ingestion   - Tail CloudWatch logs for ingestion"
	@echo "  make logs-api         - Tail App Runner logs for API"
	@echo "  make run-ingestion-aws - Manually trigger ECS ingestion task"
	@echo "  make status           - Show status of deployed services"
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
	docker compose up

dev-up-d:
	docker compose up -d

dev-down:
	docker compose down

dev-clean:
	docker compose down -v

dev: dev-up-d migrate-up api-run-d frontend-dev-d
	@echo ""
	@echo "Local dev environment started:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Adminer:    http://localhost:8080"
	@echo "  - API:        http://localhost:3000"
	@echo "  - Frontend:   http://localhost:5173"
	@echo ""
	@echo "Run 'make dev-stop' to stop all services"

dev-stop: frontend-stop api-stop dev-down
	@echo "All local dev services stopped"

# ============================================================================
# Migrations
# ============================================================================

install-migrate:
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up: install-migrate
	migrate -path ./migrations -database "$(LOCAL_DB_URL)" up

migrate-down: install-migrate
	migrate -path ./migrations -database "$(LOCAL_DB_URL)" down

migrate-neon: install-migrate
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	migrate -path ./migrations -database "$(NEON_DATABASE_URL)" up

migrate-create: install-migrate
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

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
	docker build -t opscout-api:latest ./api

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
# Ingestion Service
# ============================================================================

run:
	cd ingestion && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	SAM_API_KEY="dummy-key-for-local-testing" \
	LOG_LEVEL=debug \
	go run ./cmd/ingest

run-neon:
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	@if [ -z "$(SAM_API_KEY)" ]; then \
		echo "Error: SAM_API_KEY environment variable is not set"; \
		exit 1; \
	fi
	cd ingestion && \
	DATABASE_URL="$(NEON_DATABASE_URL)" \
	SAM_API_KEY="$(SAM_API_KEY)" \
	LOG_LEVEL=debug \
	go run ./cmd/ingest

run-mock:
	cd ingestion && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	SAM_API_KEY="mock-key" \
	MOCK_API_URL="http://localhost:8080" \
	LOG_LEVEL=debug \
	RECORD_LIMIT=$(or $(RECORD_LIMIT),100) \
	go run ./cmd/ingest

run-csv-only:
	cd ingestion && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	SAM_API_KEY="dummy-key" \
	SKIP_API=true \
	SKIP_DESCRIPTIONS=true \
	LOG_LEVEL=debug \
	go run ./cmd/ingest

mock-server:
	cd ingestion && go run ./cmd/mockserver -port=8080 -count=$(or $(MOCK_COUNT),500)

# ============================================================================
# Build
# ============================================================================

build:
	cd ingestion && go build -o ../bin/ingest ./cmd/ingest

test:
	cd ingestion && go test -v ./...
	cd api && go test -v ./...

ingestion-docker-build:
	docker build -t opscout-ingestion:latest ./ingestion

# ============================================================================
# Deploy
# ============================================================================

ecr-login:
	@AWS_ACCOUNT_ID=$$(aws sts get-caller-identity --query Account --output text --profile $(AWS_PROFILE)) && \
	aws ecr get-login-password --region $(AWS_REGION) --profile $(AWS_PROFILE) | \
		docker login --username AWS --password-stdin $$AWS_ACCOUNT_ID.dkr.ecr.$(AWS_REGION).amazonaws.com

deploy-frontend: frontend-build
	@echo "Deploying frontend to S3/CloudFront..."
	@API_URL=$$(cd terraform && terraform output -raw apprunner_service_url) && \
	echo "VITE_API_URL=https://$$API_URL/api" > frontend/.env.production && \
	cd frontend && npm run build && \
	BUCKET=$$(cd ../terraform && terraform output -raw frontend_bucket_name) && \
	DIST_ID=$$(cd ../terraform && terraform output -raw cloudfront_distribution_id) && \
	aws s3 sync dist s3://$$BUCKET --delete --profile $(AWS_PROFILE) && \
	echo "Invalidating CloudFront cache..." && \
	aws cloudfront create-invalidation --distribution-id $$DIST_ID --paths "/*" --profile $(AWS_PROFILE) && \
	rm -f .env.production && \
	echo "Frontend deployed successfully!"

deploy-api: api-docker-build ecr-login
	@echo "Deploying API to App Runner..."
	@ECR_URL=$$(cd terraform && terraform output -raw ecr_api_repository_url) && \
	ARN=$$(cd terraform && terraform output -raw apprunner_service_arn) && \
	docker tag opscout-api:latest $$ECR_URL:latest && \
	docker push $$ECR_URL:latest && \
	echo "Triggering App Runner deployment..." && \
	aws apprunner start-deployment --service-arn $$ARN --profile $(AWS_PROFILE) --region $(AWS_REGION) && \
	echo "API deployment triggered! Check status with 'make status'"

deploy-ingestion: ingestion-docker-build ecr-login
	@echo "Deploying ingestion image to ECR..."
	@ECR_URL=$$(cd terraform && terraform output -raw ecr_repository_url) && \
	docker tag opscout-ingestion:latest $$ECR_URL:latest && \
	docker push $$ECR_URL:latest && \
	echo "Ingestion image pushed successfully!"

deploy-all: deploy-api deploy-ingestion deploy-frontend
	@echo ""
	@echo "All services deployed!"

# ============================================================================
# Operations
# ============================================================================

logs-ingestion:
	aws logs tail /opscout/ingestion --follow --profile $(AWS_PROFILE)

logs-api:
	@SERVICE_ARN=$$(cd terraform && terraform output -raw apprunner_service_arn) && \
	SERVICE_ID=$$(echo $$SERVICE_ARN | rev | cut -d'/' -f1 | rev) && \
	aws logs tail /aws/apprunner/opscout-api/$$SERVICE_ID/application --follow --profile $(AWS_PROFILE) --region $(AWS_REGION)

run-ingestion-aws:
	@echo "Triggering ECS ingestion task..."
	@CLUSTER=$$(cd terraform && terraform output -raw ecs_cluster_name) && \
	TASK_DEF=$$(cd terraform && terraform output -raw ecs_task_definition_arn) && \
	SUBNETS=$$(cd terraform && terraform output -json private_subnet_ids | jq -r 'join(",")') && \
	SG=$$(cd terraform && terraform output -raw security_group_id) && \
	aws ecs run-task \
		--cluster $$CLUSTER \
		--task-definition $$TASK_DEF \
		--launch-type FARGATE \
		--network-configuration "awsvpcConfiguration={subnets=[$$SUBNETS],securityGroups=[$$SG],assignPublicIp=DISABLED}" \
		--profile $(AWS_PROFILE) --region $(AWS_REGION) && \
	echo "Ingestion task triggered! Check logs with 'make logs-ingestion'"

status:
	@echo "=== App Runner API ===" && \
	ARN=$$(cd terraform && terraform output -raw apprunner_service_arn 2>/dev/null) && \
	if [ -n "$$ARN" ]; then \
		aws apprunner describe-service --service-arn $$ARN \
			--query 'Service.{Status:Status,URL:ServiceUrl,Updated:UpdatedAt}' \
			--output table --profile $(AWS_PROFILE) --region $(AWS_REGION); \
	else \
		echo "App Runner service not found (run tf-apply first)"; \
	fi
	@echo ""
	@echo "=== CloudFront Frontend ===" && \
	DIST_ID=$$(cd terraform && terraform output -raw cloudfront_distribution_id 2>/dev/null) && \
	if [ -n "$$DIST_ID" ]; then \
		aws cloudfront get-distribution --id $$DIST_ID \
			--query 'Distribution.{Status:Status,DomainName:DomainName}' \
			--output table --profile $(AWS_PROFILE); \
	else \
		echo "CloudFront distribution not found (run tf-apply first)"; \
	fi
	@echo ""
	@echo "=== Recent Ingestion Tasks ===" && \
	CLUSTER=$$(cd terraform && terraform output -raw ecs_cluster_name 2>/dev/null) && \
	if [ -n "$$CLUSTER" ]; then \
		aws ecs list-tasks --cluster $$CLUSTER \
			--family opscout-ingestion --desired-status STOPPED --max-items 5 \
			--query 'taskArns' --output table --profile $(AWS_PROFILE) --region $(AWS_REGION) 2>/dev/null || \
			echo "No recent tasks"; \
	else \
		echo "ECS cluster not found (run tf-apply first)"; \
	fi

# ============================================================================
# Terraform
# ============================================================================

tf-init:
	cd terraform && terraform init

tf-plan:
	cd terraform && terraform plan

tf-apply:
	cd terraform && terraform apply

tf-output:
	cd terraform && terraform output

tf-destroy:
	cd terraform && terraform destroy

tf-fmt:
	cd terraform && terraform fmt

tf-validate:
	cd terraform && terraform validate

# ============================================================================
# Cleanup
# ============================================================================

clean:
	rm -rf bin/
	rm -rf .logs/
	rm -rf frontend/node_modules frontend/dist
	docker compose down -v
