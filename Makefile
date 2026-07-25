# Makefile for golang-base
# Go + Fiber + Bun ORM Project

# App info
APP_NAME=golang-base
MIGRATE_BIN=./bin/migrate
SEED_BIN=./bin/seed

# Build directories
BIN_DIR=./bin
CMD_API=./cmd/api
CMD_MIGRATE=./cmd/migrate
CMD_SEED=./cmd/seed

.PHONY: help build build-api build-migrate build-seed run run-dev tidy clean migrate-up migrate-down migrate-create migrate-list migrate-fresh test seed seed-list seed-all

# Default target
help: ## Show this help message
	@echo ""
	@echo "Available commands for $(APP_NAME):"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  make \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

## Build all binaries (API + Migrate)
build: build-api build-migrate
	@echo ""
	@echo "Build complete!"
	@ls -lh $(BIN_DIR)/$(APP_NAME) $(BIN_DIR)/migrate 2>/dev/null || true

## Build API server
build-api: ## Build the API server binary
	@echo "Building API server..."
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_API)/main.go
	@echo "API server built: $(BIN_DIR)/$(APP_NAME)"

## Build migration CLI tool
build-migrate: ## Build the migration CLI tool
	@echo "Building migrate CLI..."
	go build -o $(MIGRATE_BIN) $(CMD_MIGRATE)/main.go
	@echo "Migration tool built: $(MIGRATE_BIN)"

## Build seeder CLI tool
build-seed: ## Build the seeder CLI tool
	@echo "Building seed CLI..."
	go build -o $(SEED_BIN) $(CMD_SEED)/main.go
	@echo "Seeder tool built: $(SEED_BIN)"

## Run API server
run: build-api
	@echo "Starting $(APP_NAME)..."
	$(BIN_DIR)/$(APP_NAME)

## Run API server with hot-reload (requires air)
run-dev: build-api
	@echo "Starting $(APP_NAME) in dev mode..."
	@if command -v air > /dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "'air' not found. Install with: go install github.com/cosmtrek/air@latest"; \
		$(BIN_DIR)/$(APP_NAME); \
	fi

## Tidy Go modules
tidy: ## Download and clean Go dependencies
	go mod tidy
	@echo "Dependencies tidied"

## Clean build artifacts
clean: ## Remove all build artifacts and binaries
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "Clean complete"

## Run database migrations (up)
migrate-up: build-migrate ## Run all pending migrations
	@echo "Running migrations..."
	$(MIGRATE_BIN) up

## Seed specific seeder (usage: make seed NAME=SuperAdminSeeder ARGS="email pass role")
seed: build-seed
	@echo "Running seeder: $(NAME)"
	./bin/seed $(NAME) $(ARGS)

## List all available seeders
seed-list: build-seed
	@./bin/seed --list

## Run all seeders (use ARGS=--force to force re-seed)
seed-all: build-seed
	@echo "Running all seeders..."
	@./bin/seed $(ARGS)

## Rollback last migration
migrate-down: build-migrate ## Rollback the last migration
	@echo "Rolling back last migration..."
	$(MIGRATE_BIN) down

## Create new migration file
migrate-create: build-migrate ## Create a new migration (usage: make migrate-create name=add_users_table)
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=add_users_table"; \
		exit 1; \
	fi
	@echo "Creating migration: $(name)"
	$(MIGRATE_BIN) create $(name)

## List all migrations and their status
migrate-list: build-migrate ## List all migrations with status
	@echo "Listing migrations..."
	$(MIGRATE_BIN) list

## Drop all tables and re-run all migrations
migrate-fresh: build-migrate ## Drop all tables and re-run all migrations
	@echo "Running fresh migration..."
	$(MIGRATE_BIN) fresh

## Run e2e integration tests (requires E2E_DATABASE_URL env var)
test-e2e: ## Run e2e integration tests with real database
	@if [ -z "$(E2E_DATABASE_URL)" ]; then echo "Error: E2E_DATABASE_URL not set"; echo "Usage: E2E_DATABASE_URL=postgres://... make test-e2e"; exit 1; fi
	@echo "Running e2e integration tests..."
	E2E_DATABASE_URL=$(E2E_DATABASE_URL) JWT_SECRET=test-secret go test -v -count=1 -timeout=180s ./e2e/integration/...

## Run unit tests (excluding integration)
test: ## Run all tests
	@echo "Running tests..."
	go test -v ./internal/... ./cmd/...

## Run tests with coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Format code
fmt: ## Format Go code
	go fmt ./...
	@echo "Code formatted"

## Lint code
lint: ## Run linter
	@echo "Linting code..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "'golangci-lint' not found. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"; \
	fi

## Show project structure
tree: ## Show project directory structure
	@find . -type f \( -name "*.go" -o -name "*.sql" -o -name "Makefile" -o -name ".env*" \) | \
		grep -v "\.git/" | sort | head -30

## Rename project module
rename-project: ## Rename the project module name (usage: make rename-project name=github.com/newuser/newproject)
	@if [ -z "$(name)" ]; then \
		echo "Usage: make rename-project name=github.com/newuser/newproject"; \
		exit 1; \
	fi
	@echo "Renaming project from $$(go list -m) to $(name)..."
	@find . -type f -not -path "*/\.git/*" -not -path "*/\.codegraph/*" -exec sed -i 's|'$$(go list -m)'|$(name)|g' {} +
	@go mod edit -module=$(name)
	@go mod tidy
	@echo "Project renamed to $(name) successfully!"

## Create a new module scaffolding
make-module: ## Scaffold a new full CRUD module (usage: make make-module name=product)
	@if [ -z "$(name)" ]; then \
		echo "Usage: make make-module name=product"; \
		exit 1; \
	fi
	@./scripts/make-module.sh $(name)

## Start local development services via Docker
docker-up: ## Start PostgreSQL and Redis via Docker Compose
	@echo "Starting local services (postgres, redis)..."
	docker-compose -f docker-compose.dev.yml up -d postgres redis

## Stop local development services
docker-down: ## Stop Docker Compose services
	@echo "Stopping local services..."
	docker-compose -f docker-compose.dev.yml down

## Create a new seeder
make-seeder: ## Scaffold a new seeder (usage: make make-seeder name=Product)
	@if [ -z "$(name)" ]; then \
		echo "Usage: make make-seeder name=Product"; \
		exit 1; \
	fi
	@echo "Scaffolding new seeder: $(name)Seeder..."
	@echo "package seeders\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"github.com/uptrace/bun\"\n)\n\ntype $(name)Seed struct{}\n\nfunc (s *$(name)Seed) Name() string { return \"$(name)Seeder\" }\n\nfunc (s *$(name)Seed) Order() int { return 10 }\n\nfunc (s *$(name)Seed) SetArgs(args []string) {}\n\nfunc (s *$(name)Seed) Run(db *bun.DB) error {\n\t// ctx := context.Background()\n\tfmt.Println(\"  Running $(name)Seeder...\")\n\treturn nil\n}\n\nfunc init() { Register(&$(name)Seed{}) }\n" > cmd/seed/seeders/$(shell echo $(name) | tr A-Z a-z).go
	@echo "Seeder $(name)Seeder created at cmd/seed/seeders/$(shell echo $(name) | tr A-Z a-z).go"
