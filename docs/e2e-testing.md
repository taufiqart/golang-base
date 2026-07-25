# E2E Integration Testing

Integration tests validate HTTP endpoints end-to-end against **real PostgreSQL** via `app.New()` (same production code), with fresh migrations on every run.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│  go test -count=1 ./e2e/integration/...                  │
│                                                          │
│  ┌────────────────┐   ┌──────────────┐   ┌───────────┐  │
│  │ TestMain()      │──▶│ freshMigrate │──▶│ DROP      │  │
│  │ (once per run)  │   │ (re-run all  │   │ SCHEMA +  │  │
│  │                 │   │  migrations) │   │ re-create │  │
│  └────────────────┘   └──────────────┘   └───────────┘  │
│                                                          │
│  ┌──────────┐  TruncateTables  ┌──────────────┐         │
│  │ Each     │─────────────────▶│ seedAdminUser │         │
│  │ Test     │                  └──────┬───────┘         │
│  │          │                         │                  │
│  │          │                         ▼                  │
│  │          │                  ┌──────────────┐         │
│  │          │◀─────────────────│ AdminToken()  │         │
│  │          │   (login via     │ (real auth    │         │
│  │          │    HTTP request) │  endpoint)    │         │
│  └──────────┘                  └──────────────┘         │
│       │                                                 │
│       ▼                                                 │
│  ┌──────────┐   ┌────────────┐   ┌──────────────────┐  │
│  │ Do()     │──▶│ Fiber      │──▶│ Real Handler →    │  │
│  │ Helper   │   │ app.Test() │   │ Service → Repo →  │  │
│  │          │   │ (HTTP req)  │   │ PostgreSQL (Bun)  │  │
│  └──────────┘   └────────────┘   └──────────────────┘  │
│       │                                                 │
│       ▼                                                 │
│  ┌──────────┐                                           │
│  │ Assert   │                                           │
│  │ Helpers  │                                           │
│  └──────────┘                                           │
└──────────────────────────────────────────────────────────┘
```

Key feature: the app is built with `app.New()` — **same production code** — running against a dedicated test database.

## File Structure

```
e2e/
└── integration/
    ├── setup_test.go                # TestMain: DB init, freshMigrate, seed helpers
    ├── helpers_test.go              # Do(), Assert*(), TruncateTables, AdminToken, newUUID, loginAs
    ├── auth_test.go                 # Login, me, refresh, invalid token (10 tests)
    ├── user_test.go                 # User CRUD, duplicate email, 404, pagination (9 tests)
    ├── item_category_test.go            # Task type CRUD, duplicate, 404, pagination (9 tests)
    ├── age_group_test.go            # Age group CRUD, duplicate, 404, pagination (8 tests)
    ├── address_test.go              # Address CRUD, country/city hierarchy, filter by level (11 tests)
    ├── document_category_test.go    # Document category CRUD, duplicate name (9 tests)
    ├── document_type_test.go        # Document type CRUD, FK validation, category relation (10 tests)
    ├── document_group_test.go       # Document group CRUD, items, replace items, logic validation (12 tests)
    ├── country_group_test.go        # Country group CRUD, add/remove members, duplicate code (12 tests)
    ├── permission_test.go           # Permission matrix, roles, grant/revoke, audit logs (14 tests)
    └── permission_middleware_test.go # Forbidden vs authorized access, super admin bypass (22 tests)
```

## Running Tests

```bash
# Via Makefile (recommended)
E2E_DATABASE_URL=postgres://user:pass@localhost:5432/golang_base_test make test-e2e

# Or directly:
E2E_DATABASE_URL=postgres://user:pass@localhost:5432/golang_base_test \
  JWT_SECRET=test-secret \
  go test -v -count=1 -timeout=180s ./e2e/integration/...

# Run specific test
E2E_DATABASE_URL=postgres://user:pass@localhost:5432/golang_base_test \
  JWT_SECRET=test-secret \
  go test -v -count=1 -timeout=180s ./e2e/integration/... -run TestCreateUser_Success
```

## Environment Variables

| Variable           | Purpose                                          |
| ------------------ | ------------------------------------------------ |
| `E2E_DATABASE_URL` | PostgreSQL connection string for test DB         |
| `JWT_SECRET`       | JWT signing secret (use `test-secret` for tests) |

An `.env` file with `E2E_DATABASE_URL` is automatically loaded if present.

## Key Helpers

### Request/Response

| Helper                         | Description                                                                     |
| ------------------------------ | ------------------------------------------------------------------------------- |
| `Do(t, app, req)`              | Execute HTTP request via Fiber's `app.Test()` — real routing/middleware/handler |
| `AssertOK(t, resp)`            | Assert 200 OK                                                                   |
| `AssertCreated(t, resp)`       | Assert 201 Created                                                              |
| `AssertBadRequest(t, resp)`    | Assert 400 Bad Request                                                          |
| `AssertUnauthorized(t, resp)`  | Assert 401 Unauthorized                                                         |
| `AssertForbidden(t, resp)`     | Assert 403 Forbidden                                                            |
| `AssertNotFound(t, resp)`      | Assert 404 Not Found                                                            |
| `AssertInternalError(t, resp)` | Assert 500 Internal Server Error                                                |

### Response Body

| Helper                           | Description                                           |
| -------------------------------- | ----------------------------------------------------- |
| `AssertDataEnvelope(t, resp)`    | Assert `{"data": {...}}` and return the `data` object |
| `AssertDataList(t, resp)`        | Assert `{"data": [...]}` and return the `data` array  |
| `AssertMessageEnvelope(t, resp)` | Assert `{"message": "..."}`                           |
| `AssertPaginationMeta(t, resp)`  | Assert `{"meta": {...}}` and return the `meta` object |

### Setup

| Helper                              | Description                                                  |
| ----------------------------------- | ------------------------------------------------------------ |
| `TruncateTables(t, tables...)`      | `TRUNCATE ... CASCADE` specified tables (resets sequences)   |
| `seedAdminUser(t)`                  | Insert super_admin user into DB                              |
| `AdminToken(t)`                     | Login as admin@test.com via real HTTP endpoint, return token |
| `seedGuestUser(t)`                  | Create guest user + return token (for permission tests)      |
| `seedRolePermission(t, role, perm)` | Insert role_permission row                                   |
| `newUUID(t)`                        | Generate UUID v7 string                                      |
| `loginAs(t, email, pass)`           | Login via real HTTP endpoint, return token                   |
| `getAdminID(t, token)`              | Fetch admin user ID from list endpoint                       |

## Writing Tests

### Pattern

Every test follows this structure:

1. **`TruncateTables(t, "users", "role_permissions", ...)`** — clean relevant tables
2. **`seedAdminUser(t)`** — seed super_admin user
3. **`token := AdminToken(t)`** — login via real auth endpoint
4. **Execute request** — `Do(t, TestApp, Request{...})`
5. **Assert response** — status, envelope, fields

### Example

```go
func TestCreateItemCategory_Success(t *testing.T) {
    TruncateTables(t, "users", "item_categories", "role_permissions")
    seedAdminUser(t)
    token := AdminToken(t)

    resp := Do(t, TestApp, Request{
        Method: "POST",
        Path:   "/api/v1/masters/item-categories",
        Token:  token,
        Body:   map[string]string{"name": "Student Item", "slug": "student-item"},
    })
    AssertCreated(t, resp)

    data := AssertDataEnvelope(t, resp)
    assert.Equal(t, "Student Item", data["name"])
    assert.Equal(t, "student-item", data["slug"])
}
```

### Permission Middleware Tests

For endpoints protected by `AuthMiddleware()` + `AllowedPermissions()`:

```go
func TestCreateUser_Forbidden(t *testing.T) {
    TruncateTables(t, "users", "role_permissions")
    guestToken := seedGuestUser(t) // guest role has no permissions

    resp := Do(t, TestApp, Request{
        Method: "POST",
        Path:   "/api/v1/users",
        Token:  guestToken,
        Body:   map[string]string{"email": "nope@test.com", "name": "Nope", "password": "Test@1234", "roles": []string{"staff"}},
    })
    AssertForbidden(t, resp)
}

func TestCreateUser_Authorized(t *testing.T) {
    TruncateTables(t, "users", "role_permissions")
    guestToken := seedGuestUser(t)
    seedRolePermission(t, "guest", "user.create") // grant permission

    resp := Do(t, TestApp, Request{
        Method: "POST",
        Path:   "/api/v1/users",
        Token:  guestToken,
        Body:   map[string]string{"email": "ok@test.com", "name": "OK", "password": "Test@1234", "roles": []string{"staff"}},
    })
    AssertCreated(t, resp) // not forbidden anymore
}
```

### Pagination

```go
func TestListItemCategorys_Pagination(t *testing.T) {
    TruncateTables(t, "users", "item_categories", "role_permissions")
    seedAdminUser(t)
    token := AdminToken(t)

    // Seed data via API
    for i := 0; i < 2; i++ {
        Do(t, TestApp, Request{
            Method: "POST", Path: "/api/v1/masters/item-categories", Token: token,
            Body: map[string]string{
                "name": fmt.Sprintf("Task %d", i+1),
                "slug": fmt.Sprintf("task-%d", i+1),
            },
        })
    }

    resp := Do(t, TestApp, Request{
        Method: "GET",
        Path:   "/api/v1/masters/item-categories?page=1&limit=10",
        Token:  token,
    })
    AssertOK(t, resp)

    meta := AssertPaginationMeta(t, resp)
    assert.Equal(t, float64(2), meta["total_items"])
    assert.Equal(t, float64(1), meta["current_page"])
}
```

## Database Isolation

| Layer                      | Mechanism                                                                                             |
| -------------------------- | ----------------------------------------------------------------------------------------------------- |
| **Between test runs**      | `freshMigrate()` — `DROP SCHEMA public CASCADE` + re-run all migrations once before all tests         |
| **Between test functions** | `TruncateTables(t, ...)` — `TRUNCATE ... CASCADE` on specific tables, resets auto-increment sequences |
| **Per-test seed**          | Each test seeds its own data (admin user, master data, etc.)                                          |

## Redis

Redis is **not required** for integration tests. The service degrades gracefully:

- `database.Redis` is set to `nil`
- Repository methods fall back to direct DB queries when cache is unavailable

## Adding a New Module Test

1. **Create `e2e/integration/<module>_test.go`** in `package integration`
2. **Import** `golang-base/internal/database` and `golang-base/internal/domain` if needed
3. **Follow the pattern**: `TruncateTables` → `seedAdminUser` → `AdminToken` → `Do()` → assertions
4. **Cover**: success, duplicate/conflict, missing fields, not found (404), pagination, update, delete
5. **If permission-protected**: add forbidden test using `seedGuestUser` + unauthorized test without token
6. **Run**: `E2E_DATABASE_URL=... JWT_SECRET=test-secret make test-e2e`
