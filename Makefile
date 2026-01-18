.PHONY: help dev-up dev-up-d dev-down migrate-up migrate-down migrate-create run run-neon run-mock run-csv-only mock-server build test clean tf-init tf-plan tf-apply docker-build docker-push api-run api-run-d api-stop api-build frontend-install frontend-dev frontend-dev-d frontend-stop frontend-build dev dev-stop

# Default target
help:
	@echo "OpScout Development Commands"
	@echo ""
	@echo "Local Development:"
	@echo "  make dev           - Start everything (db + api + frontend) in background"
	@echo "  make dev-stop      - Stop all local dev services"
	@echo "  make dev-up        - Start local PostgreSQL + Adminer (foreground)"
	@echo "  make dev-up-d      - Start local PostgreSQL + Adminer (detached)"
	@echo "  make dev-down      - Stop local database services"
	@echo ""
	@echo "Search Client (API + Frontend):"
	@echo "  make api-run       - Run API server locally (localhost:3000)"
	@echo "  make api-run-d     - Run API server in background"
	@echo "  make api-stop      - Stop background API server"
	@echo "  make api-build     - Build API binary"
	@echo "  make frontend-dev  - Run frontend dev server (localhost:5173)"
	@echo "  make frontend-dev-d - Run frontend dev server in background"
	@echo "  make frontend-stop - Stop background frontend server"
	@echo "  make frontend-build - Build frontend for production"
	@echo ""
	@echo "Ingestion Service:"
	@echo "  make run           - Run ingestion service locally (local DB)"
	@echo "  make run-neon      - Run ingestion service locally (Neon DB)"
	@echo "  make run-mock      - Run ingestion against mock SAM.gov server"
	@echo "  make run-csv-only  - Run CSV-only ingestion (skip API)"
	@echo "  make mock-server   - Start mock SAM.gov API server"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up    - Run migrations (local DB)"
	@echo "  make migrate-down  - Rollback migrations (local DB)"
	@echo "  make migrate-neon  - Run migrations (Neon DB)"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build ingestion binary"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make test          - Run tests"
	@echo ""
	@echo "Terraform:"
	@echo "  make tf-init       - Initialize Terraform"
	@echo "  make tf-plan       - Plan Terraform changes"
	@echo ""

# ============================================================================
# Local Development
# ============================================================================

# Local database URL
LOCAL_DB_URL := postgres://opscout:localdev@localhost:5432/opscout?sslmode=disable

dev-up:
	docker compose up

dev-up-d:
	docker compose up -d

dev-down:
	docker compose down

dev-clean:
	docker compose down -v

# Start everything for local development
dev: dev-up-d migrate-up api-run-d frontend-dev-d
	@echo ""
	@echo "Local dev environment started:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Adminer:    http://localhost:8080"
	@echo "  - API:        http://localhost:3000"
	@echo "  - Frontend:   http://localhost:5173"
	@echo ""
	@echo "Run 'make dev-stop' to stop all services"

# Stop all local dev services
dev-stop: frontend-stop api-stop dev-down
	@echo "All local dev services stopped"

# ============================================================================
# Migrations (using golang-migrate)
# ============================================================================

# Install golang-migrate if not present
install-migrate:
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations on local database
migrate-up: install-migrate
	migrate -path ./migrations -database "$(LOCAL_DB_URL)" up

# Rollback migrations on local database
migrate-down: install-migrate
	migrate -path ./migrations -database "$(LOCAL_DB_URL)" down

# Run migrations on Neon (requires NEON_DATABASE_URL env var)
migrate-neon: install-migrate
	@if [ -z "$(NEON_DATABASE_URL)" ]; then \
		echo "Error: NEON_DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	migrate -path ./migrations -database "$(NEON_DATABASE_URL)" up

# Create a new migration
migrate-create: install-migrate
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

# ============================================================================
# API Service (Search Client)
# ============================================================================

# Run the API server locally
api-run:
	cd api && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	LOG_LEVEL=debug \
	go run ./cmd/api

# Run the API server in background
api-run-d:
	@mkdir -p .logs
	@echo "Starting API server in background..."
	@cd api && DATABASE_URL="$(LOCAL_DB_URL)" LOG_LEVEL=debug \
		nohup go run ./cmd/api > ../.logs/api.log 2>&1 & echo $$! > ../.logs/api.pid
	@sleep 3
	@echo "API server started (PID: $$(cat .logs/api.pid))"
	@echo "Logs: .logs/api.log"

# Stop background API server
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

# Build the API binary
api-build:
	cd api && go build -o ../bin/api ./cmd/api

# ============================================================================
# Frontend
# ============================================================================

# Install frontend dependencies
frontend-install:
	cd frontend && npm install

# Run frontend dev server
frontend-dev:
	cd frontend && npm run dev

# Run frontend dev server in background
frontend-dev-d:
	@mkdir -p .logs
	@echo "Starting frontend dev server in background..."
	@(cd frontend && nohup npm run dev > ../.logs/frontend.log 2>&1) & echo $$! > .logs/frontend.pid
	@sleep 2
	@echo "Frontend dev server started (PID: $$(cat .logs/frontend.pid))"
	@echo "Logs: .logs/frontend.log"

# Stop background frontend server
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

# Build frontend for production
frontend-build:
	cd frontend && npm run build

# ============================================================================
# Ingestion Service
# ============================================================================

# Run the ingestion service locally against local PostgreSQL
run:
	cd ingestion && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	SAM_API_KEY="dummy-key-for-local-testing" \
	LOG_LEVEL=debug \
	go run ./cmd/ingest

# Run the ingestion service locally against Neon
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

# Run ingestion against mock SAM.gov server (start mock-server first)
run-mock:
	cd ingestion && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	SAM_API_KEY="mock-key" \
	MOCK_API_URL="http://localhost:8080" \
	LOG_LEVEL=debug \
	RECORD_LIMIT=$(or $(RECORD_LIMIT),100) \
	go run ./cmd/ingest

# Run CSV-only ingestion (skip API and descriptions)
run-csv-only:
	cd ingestion && \
	DATABASE_URL="$(LOCAL_DB_URL)" \
	SAM_API_KEY="dummy-key" \
	SKIP_API=true \
	SKIP_DESCRIPTIONS=true \
	LOG_LEVEL=debug \
	go run ./cmd/ingest

# Start mock SAM.gov API server
mock-server:
	cd ingestion && go run ./cmd/mockserver -port=8080 -count=$(or $(MOCK_COUNT),500)

# ============================================================================
# Build
# ============================================================================

# Build the Go binary
build:
	cd ingestion && go build -o ../bin/ingest ./cmd/ingest

# Run tests
test:
	cd ingestion && go test -v ./...
	cd api && go test -v ./...

# Build Docker image
docker-build:
	docker build -t opscout-ingestion:latest ./ingestion

# ============================================================================
# Terraform
# ============================================================================

tf-init:
	cd terraform && terraform init

tf-plan:
	cd terraform && terraform plan

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
