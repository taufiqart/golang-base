# Golang Base Template - Project Structure

Dokumen ini berisi spesifikasi arsitektur dan struktur folder untuk template ini. Struktur Modular dengan Clean Architecture, Fiber v2, Bun ORM, dan PostgreSQL.

---

## Project Structure

```
golang-base/
├── cmd/
│   ├── api/
│   │   └── main.go              # Entry point utama (Dependency Injection & server boot)
│   ├── migrate/
│   │   └── main.go              # CLI untuk database migrations
│   └── seed/
│       ├── main.go              # CLI untuk database seeders
│       └── seeders/
│           ├── seeder.go        # Seeder interface
│           ├── registry.go      # Seeder registry
│           ├── role_permissions.go  # Role permissions seed data
│           └── super_admin.go   # Super admin seed data
├── config/
│   └── config.go                # Load environment variables & database connection
├── internal/
│   ├── app/
│   │   └── app.go               # Application bootstrap & DI wiring (for testability)
│   ├── database/                # Database connection setup
│   │   ├── postgres.go          # PostgreSQL (Bun ORM)
│   │   └── redis.go             # Redis client
│   ├── domain/                  # Shared domain layer (entities, interfaces, constants, errors)
│   │   ├── user.go              # User entity
│   │   ├── role.go              # Role, RolePermission, UserPermission entities
│   │   ├── permissions.go       # Permission & role constants, definitions
│   │   ├── cache.go             # Cache key constants
│   │   ├── interfaces.go        # Repository & service interfaces
│   │   └── errors.go            # Domain error definitions
│   ├── e2e/                     # End-to-end tests
│   │   ├── e2e_test.go          # API integration tests
│   │   └── openapi_test.go      # OpenAPI compliance tests
│   ├── middleware/              # HTTP middleware
│   │   ├── middleware.go        # Global middleware (CORS, Logger, Recover)
│   │   ├── permission.go        # Permission-based access control
│   │   └── permission_test.go
│   ├── modules/                 # Feature modules (Clean Architecture)
│   │   ├── auth/
│   │   │   ├── dto.go           # Request/Response DTOs
│   │   │   ├── dto_test.go
│   │   │   ├── handler.go       # HTTP handlers
│   │   │   ├── service.go       # Business logic
│   │   │   ├── service_test.go
│   │   │   ├── repository.go    # Data access (Bun ORM queries)
│   │   │   └── module.go        # Module init & route registration
│   │   └── user/
│   │       ├── dto.go           # Request/Response DTOs
│   │       ├── dto_test.go
│   │       ├── handler.go       # HTTP handlers
│   │       ├── service.go       # Business logic
│   │       ├── service_test.go
│   │       ├── repository.go    # Data access (Bun ORM queries)
│   │       └── module.go        # Module init & route registration
│   └── pkg/                     # Shared utilities
│       ├── cache/               # Generic Redis cache manager (Remember, Get, Set)
│       │   └── cache.go
│       ├── event/               # Event Dispatcher / Pub-Sub
│       │   └── event.go
│       ├── jwt/                 # JWT token generation & validation
│       │   ├── jwt.go
│       │   └── jwt_test.go
│       ├── mailer/              # SMTP mail delivery abstraction
│       │   ├── mailer.go
│       │   └── gomail.go
│       ├── mapper/              # Object mapper (DTO transformer)
│       │   └── mapper.go
│       ├── queue/               # Background task queue (Asynq/Redis)
│       │   ├── queue.go
│       │   └── asynq.go
│       ├── storage/             # Cloud/Local file storage abstraction
│       │   ├── local.go
│       │   ├── s3.go
│       │   └── gcs.go
│       └── response/            # Standardized API response helpers
│           ├── response.go
│           ├── response_test.go
│           └── pagination.go
├── migrations/                  # SQL migrations (up/down scripts)
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   ├── 002_create_role_permissions.up.sql
│   ├── 002_create_role_permissions.down.sql
│   ├── 003_create_user_permissions.up.sql
│   ├── 003_create_user_permissions.down.sql
│   ├── 004_create_permission_changes_log.up.sql
│   └── 004_create_permission_changes_log.down.sql
├── docs/                        # Dokumentasi
│   ├── structure.md
│   └── blueprint/               # API specs & OpenAPI
│       ├── openapi.yml
│       ├── user.md
│       └── auth.md
├── scratch/                     # Temp/test files (git-ignored)
├── bin/                         # Built binaries (git-ignored)
├── Makefile                     # Build automation
├── go.mod                       # Go module dependencies
├── go.sum                       # Checksums
├── .air.toml                    # Air hot-reload config
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
   PostgreSQL
```

---

## Domain Layer

Semua domain entities, interfaces, constants, dan errors didefinisikan di `internal/domain/`:

- **Entities**: User, Role, RolePermission, UserPermission, PermissionChangesLog
- **Interfaces**: UserRepository, RolePermissionRepository, UserService, AuthService, dll.
- **Constants**: Permission constants, role constants, cache keys
- **Errors**: ErrUserNotFound, ErrUserExists, ErrInvalidCredentials, dll.

Rule: modul import `internal/domain/` untuk semua domain types. Tidak ada `domain.go` di modul.

---

## Module Structure

Setiap modul (e.g., `user`, `auth`) berada di `internal/modules/<name>/` dengan struktur:

```
<module>/
├── dto.go             # Request/Response structs
├── dto_test.go
├── handler.go         # HTTP handlers (presentation layer)
├── service.go         # Business logic (application layer)
├── service_test.go
├── repository.go      # Data access (infrastructure layer)
├── module.go          # Dependency wiring & route registration
└── [optional]
    └── cache.go       # Cache layer (if needed)
```

Domain types (entities, interfaces, constants) ada di `internal/domain/`, bukan di modul.

---

## Dependency Flow (Anti-Loop Import)

- **Domain** (`internal/domain/`): All entities, interfaces, constants, errors. No dependencies.
- **Repository**: Implements repository interface from domain. Depends on `database/`.
- **Service**: Implements service interface from domain. Depends on repository interfaces.
- **Handler**: Depends on service interfaces. No business logic.

**Rule**: Higher-level modules depend on interfaces in lower-level modules. Never import concrete implementations across modules.

---

## Stack

| Component      | Technology       |
| -------------- | ---------------- |
| Language       | Go 1.26.2        |
| HTTP Framework | Fiber v2         |
| ORM            | Bun              |
| Database       | PostgreSQL       |
| Cache          | Redis            |
| Migrations     | urfave/cli       |
| Auth           | JWT (golang-jwt) |
| Hot Reload     | Air              |

---

## Quick Commands

| Command                             | Description                               |
| ----------------------------------- | ----------------------------------------- |
| `make run`                          | Build & start API server                  |
| `make run-dev`                      | Start with hot-reload (air)               |
| `make build`                        | Build all binaries (API + migrate + seed) |
| `make migrate-up`                   | Run pending migrations                    |
| `make migrate-down`                 | Rollback last migration                   |
| `make migrate-create name=xxx`      | Create new migration                      |
| `make seed NAME=xxx`                | Run specific seeder                       |
| `make seed-all`                     | Run all seeders                           |
| `make seed-all ARGS="--except=xxx"` | Run all seeders except the specified ones |
| `make test`                         | Run all tests                             |
| `make test-coverage`                | Run tests with coverage report            |
| `make clean`                        | Remove build artifacts                    |

---

## Seeders

Semua file seeder (di `cmd/seed/seeders/`) **DIWAJIBKAN** untuk berinteraksi dengan database melalui layer **Service** atau **Repository** (misalnya `auth.NewRepository(db)`).
Penggunaan raw query SQL secara langsung seperti `INSERT INTO`, `UPDATE`, atau manipulasi ORM murni tanpa melalui repository sangat dilarang agar validasi bisnis dan logika aplikasi tetap konsisten.

---

## Adding New Module

1. **Create module folder**: `internal/modules/<name>/`
2. **Define domain**: `domain.go` - Entity struct & interfaces
3. **Implement layers**:
   - `repository.go` - Bun queries
   - `service.go` - Business logic
   - `handler.go` - HTTP handlers
   - `dto.go` - Request/Response DTOs
4. **Wire in module.go**: Register routes & inject dependencies
5. **Register in main.go**: Call `<module>.New().Register(apiGroup)`
6. **Add migration** in `migrations/`

---

## Configuration

Environment variables (via `.env`):

```env
# Application
PORT=3100
NODE_ENV=development

# Database
DATABASE_URL=postgres://root:root@localhost:5432/pose_travel_restore?sslmode=disable

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# Auth
JWT_SECRET=your-secret-key
```
