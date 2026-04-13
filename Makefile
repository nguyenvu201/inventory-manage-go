# ══════════════════════════════════════════════════════════════════
# Inventory Management System — Makefile
# Standards: FDA 21 CFR Part 11 / IEC 62304
# ══════════════════════════════════════════════════════════════════

BINARY_NAME   := inventory-manage
BUILD_DIR     := bin
CMD_PATH      := ./cmd/server
MIGRATIONS_DIR := migrations
DB_URL        ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# Load .env if present (for local dev)
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: all build run clean \
        test test-race test-integration test-smoke test-e2e test-cover test-all \
        lint \
        migrate migrate-down migrate-create migrate-status \
        docker-up docker-down docker-logs docker-build \
        help

all: lint test build  ## Run lint, test, and build

# ── Build ──────────────────────────────────────────────────────────────────────
build:  ## Build the binary to ./bin/inventory-manage
	@echo "▶ Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

run:  ## Run the service locally (loads .env automatically)
	@echo "▶ Starting service..."
	go run $(CMD_PATH)/main.go

# ── Test ───────────────────────────────────────────────────────────────────────
test:  ## Run unit tests (no external dependencies required)
	@echo "▶ Running unit tests..."
	go test ./... -v -count=1 -timeout 60s

test-race:  ## Run all tests with race detector (required before every PR)
	@echo "▶ Running tests with race detector..."
	go test -race -count=1 ./... -timeout 120s

test-integration:  ## Run integration tests — requires Docker (testcontainers-go)
	@echo "▶ Running integration tests..."
	go test -tags integration ./... -v -count=1 -timeout 180s

test-smoke:  ## Run smoke tests — requires: make docker-up + make run
	@echo "▶ Running smoke tests against live environment..."
	go test -tags smoke ./tests/smoke/... -v -count=1 -timeout 30s

test-e2e:  ## Run E2E tests — requires: full docker env + app running
	@echo "▶ Running E2E tests..."
	go test -tags e2e ./tests/e2e/... -v -count=1 -timeout 120s

test-cover:  ## Coverage report — FDA minimum: domain/ ≥90%, usecase/ ≥85%, others ≥80%
	@echo "▶ Running tests with coverage..."
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"
	go tool cover -func=coverage.out | tail -1

test-all:  ## FDA Full Test Suite: unit + race + integration + smoke + e2e + cover
	@echo "▶ FDA Full Test Suite starting..."
	$(MAKE) test
	$(MAKE) test-race
	$(MAKE) test-integration
	$(MAKE) test-smoke
	$(MAKE) test-e2e
	$(MAKE) test-cover
	@echo "✓ All test suites passed — ready for QA review"

# ── Lint ───────────────────────────────────────────────────────────────────────
lint:  ## Run go vet and staticcheck
	@echo "▶ Running go vet..."
	go vet ./...
	@echo "▶ Running staticcheck..."
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || \
		echo "⚠ staticcheck not installed. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"

swag:  ## Generate Swagger documentation
	@echo "▶ Generating Swagger docs..."
	swag init -g cmd/server/main.go --parseDependency --parseInternal
	@echo "✓ Swagger docs generated in docs/"

# ── Migrations ─────────────────────────────────────────────────────────────────
migrate:  ## Apply all pending migrations (up)
	@echo "▶ Running migrations (up)..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up
	@echo "✓ Migrations applied"

migrate-down:  ## Roll back the last migration
	@echo "▶ Rolling back last migration..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-status:  ## Show current migration version
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

migrate-create:  ## Create a new migration pair. Usage: make migrate-create NAME=create_devices_table
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=<migration_name>"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
	@echo "✓ Migration files created in $(MIGRATIONS_DIR)/"

# ── Docker ─────────────────────────────────────────────────────────────────────
docker-up:  ## Start all services via docker-compose (detached)
	@echo "▶ Starting Docker services..."
	docker-compose up -d
	@echo "✓ Services started. Run 'make docker-logs' to view logs"

docker-down:  ## Stop and remove all containers
	@echo "▶ Stopping Docker services..."
	docker-compose down

docker-logs:  ## Tail logs from all containers
	docker-compose logs -f

docker-build:  ## Rebuild Docker images
	docker-compose build --no-cache

# ── Cleanup ────────────────────────────────────────────────────────────────────
clean:  ## Remove build artifacts
	@echo "▶ Cleaning..."
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "✓ Clean complete"

# ── Help ───────────────────────────────────────────────────────────────────────
help:  ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
