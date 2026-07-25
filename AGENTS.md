# Golang Base Template - Agent Guidelines

## Project Overview

A clean, modular Golang base template (Starter Kit) for microservice development. Built with Clean Architecture principles and domain-driven design.

**Stack**: Go 1.26.2 | Fiber v2 | Bun ORM | PostgreSQL | Redis

---

## Architecture

```
golang-base/
├── cmd/                          # Application entry points
│   ├── api/main.go              # HTTP API server (DI & boot)
│   └── migrate/main.go          # Database migrations CLI
├── config/
│   └── config.go                # Environment variables loader
├── internal/
│   ├── database/                # Database connection setup
│   │   ├── postgres.go         # PostgreSQL (Bun ORM)
│   │   └── redis.go            # Redis client
│   ├── domain/                  # Shared domain (entities, interfaces, constants, errors)
│   │   ├── user.go             # User entity
│   │   ├── role.go             # Role & permission entities
│   │   ├── permissions.go      # Permission & role constants
│   │   ├── cache.go            # Cache key constants
│   │   ├── interfaces.go       # Repository & service interfaces
│   │   └── errors.go           # Domain error definitions
│   ├── middleware/
│   │   └── middleware.go        # Global middleware (CORS, Logger, Recover)
│   ├── modules/                 # Feature modules (Clean Architecture)
│   │   └── user/
│   │       ├── dto.go          # Request/Response DTOs
│   │       ├── handler.go      # HTTP handlers (presentation)
│   │       ├── service.go      # Business logic (application)
│   │       ├── repository.go   # Data access (infrastructure)
│   │       └── module.go       # Module init & route registration
│   └── pkg/                     # Shared utilities
│       ├── cache/cache.go       # Generic Redis cache manager (Remember, Get, Set)
│       ├── event/event.go       # Event Dispatcher (Pub-Sub)
│       ├── jwt/jwt.go           # JWT token generation & validation
│       ├── mailer/mailer.go     # SMTP mail delivery abstraction
│       ├── mapper/mapper.go     # Object mapper (DTO transformer)
│       ├── queue/queue.go       # Background task queue (Asynq/Redis)
│       ├── storage/local.go     # Cloud/Local file storage abstraction
│       └── response/response.go # Standardized API response helpers
├── migrations/                  # SQL migrations (up/down)
├── docs/                        # Documentation & blueprints
│   ├── structure.md
│   └── blueprint/              # Master data specs & OpenAPI
├── scratch/                     # Temp/test files (git-ignored)
├── bin/                         # Built binaries (git-ignored)
├── Makefile                     # Build automation
├── go.mod / go.sum
├── .env                         # Environment variables
└── .gitignore
```

---

## Clean Architecture Flow

```
HTTP Request
    │
    ▼
┌─────────────────┐
│   Handler       │  ← Receives HTTP, validates input, formats output
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Service       │  ← Business logic, orchestration, error mapping
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Repository    │  ← Database queries via Bun ORM
└────────┬────────┘
         │
         ▼
   PostgreSQL / Redis
```

---

## Coding Conventions

### 1. Domain Layer

- **MANDATORY UUIDv7**: All entities MUST use UUIDv7 for their primary keys (`id` fields). Do not use auto-incrementing integers (serial) or standard UUIDv4.
- Define entities as structs with Bun tags for ORM mapping (e.g., `bun:"id,pk,type:uuid"`).
- Define repository and service **interfaces** in domain
- Mark sensitive fields with `json:"-"` (e.g., password)

```go
type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`
    ID        string    `bun:"id,pk,type:uuid" json:"id"`
    Email     string    `bun:"email,unique,notnull" json:"email"`
    Password  string    `bun:"password,notnull" json:"-"`
}

type UserRepository interface {
    GetByID(id string) (*User, error)
    GetByEmail(email string) (*User, error)
    Create(user *User) error
}
```

### 2. Module Layer

Each module in `internal/modules/<name>/` contains:

| File            | Responsibility                                 |
| --------------- | ---------------------------------------------- |
| `dto.go`        | Request/Response structs                       |
| `handler.go`    | HTTP handlers (presentation layer)             |
| `service.go`    | Business logic (application layer)             |
| `repository.go` | Data access via Bun ORM (infrastructure layer) |
| `module.go`     | Dependency wiring & route registration         |

Domain types (entities, interfaces, constants, errors) live in `internal/domain/`.

### 3. Module Registration

Each module self-registers via `module.go`:

```go
type Module struct{}

func New() *Module {
    return &Module{}
}

func (m *Module) Register(router fiber.Router) {
    repo := NewRepository()
    svc := NewService(repo)
    handler := NewHandler(svc)
    handler.RegisterRoutes(router)
}
```

Then in `cmd/api/main.go`:

```go
apiGroup := app.Group("/api/v1")
user.New().Register(apiGroup)
```

### 4. HTTP Response Format

Use helpers from `internal/pkg/response/`:

```go
return response.OK(c, data)           // 200
return response.Created(c, data)      // 201
return response.BadRequest(c, msg)    // 400
return response.NotFound(c, msg)      // 404
return response.Unauthorized(c, msg)  // 401
return response.InternalError(c, msg) // 500
```

### 5. Backend Enums (No Database ENUM Types)

**Never use database-level ENUM types.** All enum-like fields must be stored as `VARCHAR` in the database and enforced via Go constants in `internal/domain/`. This ensures consistency, allows migration rollbacks without type issues, and avoids lock-in to specific databases.

```go
// CORRECT: Define constants in domain layer
const (
    TaskStatusPending    = "pending"
    TaskStatusInProgress = "in_progress"
    TaskStatusCompleted  = "completed"
    TaskStatusCancelled  = "cancelled"
)

// CORRECT: Validate at service layer
func ValidateTaskStatus(s string) bool { ... }

// CORRECT: Store as plain VARCHAR in SQL migration
// status VARCHAR(20) NOT NULL

// WRONG: Never use CREATE TYPE ... AS ENUM
```

### 6. Context Propagation

Always pass `ctx` through layers:

```go
func (h *Handler) GetProfile(c *fiber.Ctx) error {
    user, err := h.service.GetProfile(c.Context(), id)
}
```

### 7. Code Formatting and Styling (MANDATORY FOR AI AGENTS)

- **ALWAYS run `make fmt`** (or `go fmt ./...`) after writing or modifying any Go code before finishing a task.
- All Go source files must strictly adhere to standard Go formatting rules. Never leave unformatted code, unused imports, or improper indentation.

---

## Adding New Modules & Seeders (MANDATORY FOR AI AGENTS)

**AI AGENTS MUST NOT MANUALLY CREATE MODULE FILES OR DIRECTORIES.**
You must strictly use the provided Makefile commands for scaffolding to ensure structural consistency.

### 1. Scaffolding a Module

To create a new module, execute:

```bash
make make-module name=<module-name>
```

This will automatically generate `dto.go`, `handler.go`, `module.go`, `repository.go`, and `service.go` with full CRUD boilerplate in `internal/modules/<module-name>/`.
After generation:

- Add domain types to `internal/domain/` (entities, interfaces, constants, errors).
- Register the module in `cmd/api/main.go`.

### 2. Scaffolding a Seeder

To create a new database seeder, execute:

```bash
make make-seeder name=<SeederName>
```

This will generate the seeder file in `cmd/seed/seeders/`. Do not manually create the seeder file.

### 3. Database Migrations

Always create migrations using the CLI tool wrapper:

```bash
make migrate-create name=<migration_name>
```

---

## Database & Cache

- **PostgreSQL**: Primary database via Bun ORM (`internal/database/postgres.go`)
- **Redis**: Cache layer (`internal/database/redis.go`)
- Both connections are optional — service degrades gracefully if unavailable

---

## Database Migrations

- SQL-based migrations in `migrations/` directory
- Naming: `001_create_users.up.sql` / `001_create_users.down.sql`
- Run via `cmd/migrate/main.go` using urfave/cli
- Commands: `make migrate-up`, `make migrate-down`, `make migrate-create name=xxx`
- **STRICT RULE**: NEVER use `INSERT INTO` or any DML data-seeding commands in migration files. Migrations are strictly for DDL (Schema changes). All data seeding must be done through dedicated seeders (via `make make-seeder`).
- **SEEDER REPOSITORY RULE**: All seeders MUST use the Service or Repository layer for data manipulation. Direct SQL operations (e.g., `db.NewRaw("INSERT INTO ...")` or `db.Exec("UPDATE ...")`) or raw Bun ORM inserts directly in the seeder are strictly prohibited.

---

## Configuration

Environment variables (loaded via `godotenv` from `.env`):

| Variable           | Description                                            | Default          |
| ------------------ | ------------------------------------------------------ | ---------------- |
| `PORT`             | Server port                                            | `3100`           |
| `DATABASE_URL`     | PostgreSQL connection string                           | -                |
| `E2E_DATABASE_URL` | PostgreSQL connection string for E2E integration tests | -                |
| `REDIS_ADDR`       | Redis address                                          | `localhost:6379` |
| `REDIS_PASSWORD`   | Redis password                                         | -                |
| `JWT_SECRET`       | JWT signing secret                                     | fallback default |

---

## Commands

| Command                             | Description                                           |
| ----------------------------------- | ----------------------------------------------------- |
| `make run`                          | Build & start API server                              |
| `make run-dev`                      | Start with hot-reload (air)                           |
| `make build`                        | Build all binaries (API + migrate)                    |
| `make migrate-up`                   | Run pending migrations                                |
| `make migrate-down`                 | Rollback last migration                               |
| `make migrate-create name=xxx`      | Create new migration                                  |
| `make make-module name=xxx`         | Scaffold a new CRUD module                            |
| `make make-seeder name=xxx`         | Scaffold a new seeder                                 |
| `make seed NAME=xxx`                | Run a specific seeder                                 |
| `make seed-all ARGS="--except=xxx"` | Run all seeders (with optional exclusion)             |
| `make test`                         | Run all tests                                         |
| `make test-e2e`                     | Run E2E integration tests (requires E2E_DATABASE_URL) |
| `make test-coverage`                | Run tests with coverage report                        |
| `make fmt`                          | Format Go code                                        |
| `make lint`                         | Run linter (golangci-lint)                            |
| `make clean`                        | Remove build artifacts                                |
| `make tidy`                         | Tidy Go modules                                       |

---

## Git Commit Guidelines

**MANDATORY FOR AI AGENTS**: All commit messages must follow the **Conventional Commits** specification.

### Format

```
<type>[optional scope]: <description>

[optional body]
```

### Allowed Types

- `feat`: A new feature (e.g., `feat(auth): add login endpoint`)
- `fix`: A bug fix (e.g., `fix(user): resolve foreign key violation`)
- `refactor`: A code change that neither fixes a bug nor adds a feature (e.g., `refactor: simplify database seeding`)
- `chore`: Routine tasks, maintenance, dependency updates (e.g., `chore: update Makefile`)
- `docs`: Documentation changes (e.g., `docs: update AGENTS.md`)
- `test`: Adding missing tests or correcting existing tests (e.g., `test(auth): add e2e for role permissions`)
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, etc.)
- `perf`: A code change that improves performance

### Rules

1. **Lowercase Type**: The type (e.g., `feat`) must be completely lowercase.
2. **Imperative Mood**: The description must use the imperative, present tense (e.g., "add feature" not "added feature" or "adds feature").
3. **No Period**: Do not end the subject line with a period.
4. **Descriptive Body**: Use the optional body to explain _what_ and _why_ (vs. _how_).

---

## Scratch Folder Rule

**All temporary, experimental, or test files MUST be stored in `scratch/` directory.** This includes:

- Test files generated for experimentation (not part of the test suite)
- CLI command output captures
- Temporary scripts or snippets
- Prototype/exploratory code

The `scratch/` folder is **git-ignored** — its contents will never be committed.

```
scratch/
├── test_api.sh           # CLI test script
├── curl_output.json      # API response capture
├── query_test.go         # Experimental Go code
└── migration_draft.sql   # Draft migration
```

**Do NOT** place scratch files in `internal/`, `cmd/`, or project root.

---

---

## E2E Integration Testing (MANDATORY FOR AI AGENTS)

**CRITICAL RULE: AI agents MUST ALWAYS write both Unit Tests and E2E Integration Tests for every new feature, service, or handler they create or modify. Code is not complete without tests.**

All integration tests run against a **real PostgreSQL** database via `E2E_DATABASE_URL` with **fresh migrations** on every run.

### Architecture

- Each test follows: `TruncateTables` → `seedAdminUser` → `AdminToken` → `Do()` (real HTTP via `app.Test()`) → assertions
- `TestMain`: fresh migrate (`DROP SCHEMA public CASCADE` + re-run all migrations) once before all tests
- Redis is not required — `database.Redis` = nil, services fall back to direct DB queries

### Coverage (~105 tests)

| Module                | File                            | Tests                                          |
| --------------------- | ------------------------------- | ---------------------------------------------- |
| Auth                  | `auth_test.go`                  | 10 (login, me, refresh, invalid token)         |
| User                  | `user_test.go`                  | 9 (CRUD, duplicate, 404, pagination)           |
| Permission            | `permission_test.go`            | 14 (matrix, roles, grant/revoke, audit logs)   |
| Permission Middleware | `permission_middleware_test.go` | 22 (forbidden, authorized, super admin bypass) |

### CI

- Run: `E2E_DATABASE_URL=postgres://... make test-e2e`

---

## Important Notes

- **STRICT MAKEFILE USAGE** — AI agents must ALWAYS use `Makefile` commands for scaffolding (`make make-module`, `make make-seeder`, `make migrate-create`). Never create module or seeder boilerplate files manually.
- **MANDATORY CODE FORMATTING** — Always run `make fmt` after modifying or creating Go code to ensure standard formatting and styling.
- **MANDATORY UUIDv7** — Always use UUIDv7 for primary keys. Never use `id serial`, `int64`, or auto-incrementing integers.
- **SEEDERS MUST USE REPOSITORIES** — Seeders MUST use the Service or Repository layer (e.g., `auth.NewRepository(db)`). Do not use raw SQL queries or raw database manipulations directly inside seeders.
- **MANDATORY GO TESTING** — You MUST always create Unit Tests (for logic/service) and E2E Integration Tests (for HTTP handlers & DB) whenever you write or modify code.
- **Backend enums only** — never use database ENUM types; define constants in `internal/domain/` and store as `VARCHAR`
- **No business logic in handlers** - delegate to service
- **No database queries in service** - delegate to repository
- **Always use context** for cancellation/timeout propagation
- **Mark secrets with `json:"-"`** to prevent exposure
- **Return typed errors** from service layer (map DB errors to domain errors)
- **Use `response` pkg** for consistent API responses
- **Module self-registration** via `module.go` — keep `main.go` clean
- **Build outputs go to `./bin/`** — all binaries (API, migrate, seed) must be built into `bin/` directory, never to project root or elsewhere

---

## Real API Testing (MANDATORY FOR AI AGENTS)

**CRITICAL RULE: WHENEVER AN AI AGENT CREATES OR MODIFIES AN API ENDPOINT, YOU MUST ALWAYS CREATE A REAL HTTP TEST SCRIPT USING PYTHON AND ALWAYS GENERATE A DETAILED MARKDOWN REPORT.**

### Location

All real test files are stored in `scratch/module/{module-name}/`.

```
scratch/module/users/
├── real_user_test_full.py                   -- Python test script
├── real-api-full-report.md                  -- Summary report (pass/fail per scenario)
├── real-api-full-payload-response-report.md -- Detailed report with payload, response, description
└── user-test-blueprint.md                   -- Blueprint test plan
```

### Real Test Rules

1. **Use E2E Database (`E2E_DATABASE_URL`)** -- The test script MUST read from `E2E_DATABASE_URL` env and set `DATABASE_URL` to the same value. Never use or connect to the real/development database (`DATABASE_URL`).
2. **Run Dedicated Server on Random Port** -- The script MUST dynamically allocate an available random port (e.g., using Python `socket` to find a free port) and spawn its own test API server instance (`./bin/golang-base`) in the background with `PORT=<random_port>`. Never hardcode or assume a fixed port.
3. **Mandatory Server Termination** -- When the test script completes (whether passing, failing, or on exception), it MUST cleanly terminate and stop the background test server process (e.g., in a `try...finally` block).
4. **Seed via direct SQL / Seeders** -- ensure fresh migration (`make migrate-fresh`) and seed master data (`make seed-all`) against `E2E_DATABASE_URL` before starting the server.
5. **TRUNCATE ... CASCADE before seed** -- ensure clean state.
6. **Test via normal HTTP requests** -- use `urllib` or `curl`, not `app.Test()`.
7. **Each scenario has a unique code** -- format `{MODULE}-{NNN}` (e.g., `USER-001`, `AUTH-003`).
8. **Each scenario has description + flow + verification** in `DESC` dict:

```python
DESC = {
    "USER-001": "**Scenario**: Create a new user.\n**Flow**: POST /api/v1/users -> service createUser -> insert DB -> return 201.\n**Verification**: status=201, ID generated.",
}
```

9. **MANDATORY REPORT GENERATION** -- The python script MUST ALWAYS generate a `.md` report file containing the complete JSON payload and JSON response for every single HTTP request made during the test.

### Report Format (Full Markdown)

The `real-api-full-payload-response-report.md` must include:

#### Setup / Seed

- HTTP login trace (token redacted)

#### Per Scenario

- **Code + Name + Status** -- e.g., `## CORE-001: CORE-001 Create individual -- PASS`
- **Description & Flow** -- from `DESC` dict
- **Verification** -- what is checked to determine pass/fail
- **HTTP Request(s)** -- every request for the scenario:

  ````
  ### Request 1: POST /api/v1/tasks -> 201

  **Payload**
  ```json
  { ... }

  **Response**
  ```json
  { ... }
  ````

#### Summary Table

- Table: `| Suite | Scenario | Result | Detail |`
- Total row: `N passed, M failed, K scenarios, H HTTP calls`

### Python Script Internals

```
results = []              # [(suite, scenario, status, detail), ...]
http_traces = []          # [{"method", "path", "payload", "status", "response"}, ...]
result_trace_ranges = []  # (start_idx, end_idx) -- maps each result to its trace range
_seed_trace_end = 0       # number of traces before first scenario
```

- `api()` -- records trace into `http_traces`
- `log()` -- records result + trace range into `result_trace_ranges`
- `_seed_trace_end` -- set after seed is done, separates setup from tests
- Report generation: iterate `results`, find `result_trace_ranges[idx]`, take trace slice, render description from `DESC`

### Audit Report

After real test completes, provide audit analysis:

1. **Total**: pass count, fail count, scenario count, HTTP call count
2. **Failing scenarios**: list with error details, root cause (implementation gap vs test bug)
3. **Implementation gaps**: for each failure caused by the service:
   - Service does not validate field X
   - Response does not match spec
   - Error handling is incorrect
4. **Recommendations**: prioritized fix steps

Example audit:

> ## Audit Report
>
> - Summary: 15 passed, 0 failed, 15 scenarios, 30 HTTP calls
> - USER-009: ValidationError for invalid email -> status 400 PASS
> - AUTH-001: Invalid token rollback works -> 401 PASS
