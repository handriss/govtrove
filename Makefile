.PHONY: help dev-up dev-down migrate-up migrate-down migrate-create run run-neon build test clean tf-init tf-plan tf-apply docker-build docker-push

# Default target
help:
	@echo "OpScout Development Commands"
	@echo ""
	@echo "Local Development:"
	@echo "  make dev-up        - Start local PostgreSQL + Adminer (foreground)"
	@echo "  make dev-down      - Stop local services"
	@echo "  make run           - Run ingestion service locally (local DB)"
	@echo "  make run-neon      - Run ingestion service locally (Neon DB)"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up    - Run migrations (local DB)"
	@echo "  make migrate-down  - Rollback migrations (local DB)"
	@echo "  make migrate-neon  - Run migrations (Neon DB)"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build Go binary"
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

dev-down:
	docker compose down

dev-clean:
	docker compose down -v

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
# Run Service
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

# ============================================================================
# Build
# ============================================================================

# Build the Go binary
build:
	cd ingestion && go build -o ../bin/ingest ./cmd/ingest

# Run tests
test:
	cd ingestion && go test -v ./...

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
	docker compose down -v
