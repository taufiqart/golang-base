# Golang Clean Architecture Template

A base starter project for microservices or backend applications.

**Stack**: Go 1.26.2 | Fiber v2 | Bun ORM | PostgreSQL | Redis

## Architecture

Built with **Clean Architecture** principles using a modular structure:

```
golang-base/
├── cmd/                          # Application entry points
│   ├── api/
│   │   └── main.go              # HTTP API server (DI & boot)
│   ├── migrate/
│   │   └── main.go              # Database migrations CLI
│   └── seed/
│       ├── main.go              # Database seeders CLI
│       └── seeders/             # Seeder implementations
├── config/
│   └── config.go                # Environment variables loader
├── internal/
│   ├── app/
│   │   └── app.go               # Application bootstrap & DI wiring
│   ├── database/
│   │   ├── postgres.go          # PostgreSQL (Bun ORM)
│   │   └── redis.go             # Redis client
│   ├── domain/                  # Shared domain layer (entities, interfaces, constants, errors)
│   │   ├── user.go              # User entity
│   │   ├── role.go              # Role & permission entities
│   │   ├── permissions.go       # Permission & role constants
│   │   ├── cache.go             # Cache key constants
│   │   ├── interfaces.go        # Repository & service interfaces
│   │   └── errors.go            # Domain error definitions
│   ├── e2e/                     # End-to-end & OpenAPI tests
│   ├── middleware/
│   │   ├── middleware.go        # Global middleware (CORS, Logger, Recover)
│   │   └── permission.go        # Permission-based access control
│   ├── modules/
│   │   ├── auth/                # Authentication & authorization
│   │   │   ├── dto.go
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   └── module.go
│   │   └── user/                # User management
│   │       ├── dto.go
│   │       ├── handler.go
│   │       ├── service.go
│   │       ├── repository.go
│   │       └── module.go
│   └── pkg/
│       ├── jwt/                 # JWT token generation & validation
│       └── response/            # Standardized API response helpers
├── migrations/                   # SQL migration files (up/down)
├── docs/                         # Documentation & blueprints
├── scratch/                      # Temp/test files (git-ignored)
├── bin/                          # Built binaries (git-ignored)
├── Makefile
├── go.mod
├── .air.toml
└── .env
```

Each module follows the Clean Architecture flow:

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

### Domain Layer

All domain entities, interfaces, constants, and errors live in `internal/domain/`:
- **Entities**: User, Role, RolePermission, UserPermission, PermissionChangesLog
- **Interfaces**: UserRepository, RolePermissionRepository, UserService, AuthService, etc.
- **Constants**: Permission constants, role constants, cache keys
- **Errors**: ErrUserNotFound, ErrUserExists, ErrInvalidCredentials, etc.

Modules import `internal/domain/` for all domain types. No `domain.go` in modules.

## Prerequisites

- Go 1.26.2+
- PostgreSQL
- Redis (optional)
- [air](https://github.com/cosmtrek/air) (optional, for hot-reload, config in `.air.toml`)

## Setup

### 1. Clone & install dependencies

```bash
git clone <repo-url>
cd golang-base
go mod tidy
```

### 2. Configure environment

Copy `.env.example` to `.env` or create a new `.env` file:

```bash
cp .env.example .env
```

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `3100` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `JWT_SECRET` | JWT signing secret | fallback default |

### 3. Run database migrations

```bash
# Run all pending migrations
make migrate-up

# Rollback the last migration
make migrate-down

# Create a new migration
make migrate-create name=add_users_table

# List all migrations and their status
make migrate-list
```

### 4. Run database seeders

```bash
# Run a specific seeder
make seed seeder=role_permissions

# Run all seeders
make seed seeder=--all
```

### 5. Run the application

```bash
# Build & start the server
make run

# Start with hot-reload (development)
make run-dev
```

The server will start at `http://localhost:3100`.

### Health check

```bash
curl http://localhost:3100/health
```

## Other Commands

| Command | Description |
|---------|-------------|
| `make build` | Build all binaries (API + migrate + seed) |
| `make test` | Run all tests |
| `make test-coverage` | Run tests with coverage report |
| `make fmt` | Format Go code |
| `make lint` | Run linter |
| `make clean` | Remove build artifacts |
| `make tidy` | Tidy Go modules |

## Documentation

Full documentation is available in the **`docs/`** directory:

- `docs/structure.md` - Complete project structure explanation
- `docs/blueprint/` - Domain logic and OpenAPI definitions

## Adding a New Module

1. Create a folder in `internal/modules/<name>/`
2. Create files: `domain.go`, `dto.go`, `repository.go`, `service.go`, `handler.go`, `module.go`
3. Register in `cmd/api/main.go`:
   ```go
   <module>.New().Register(apiGroup)
   ```
4. Add migration in `migrations/`

## License

MIT License
