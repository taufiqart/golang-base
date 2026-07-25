package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang-base/internal/database"
	"golang-base/internal/domain"
	"golang-base/internal/modules/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// seedLimitedUser creates a user with the "user" role (limited permissions)
// and returns a valid auth token.
func seedLimitedUser(t *testing.T) string {
	t.Helper()
	ctx := t.Context()

	hash, err := bcrypt.GenerateFromPassword([]byte("Test@1234"), bcrypt.MinCost)
	require.NoError(t, err)
	hashStr := string(hash)

	id, err := uuid.NewV7()
	require.NoError(t, err)

	user := &domain.User{
		ID:        id.String(),
		Email:     "limited@test.com",
		Name:      "Limited User",
		Password:  &hashStr,
		Roles:     []string{"user"},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Ensure "user" role exists before inserting
	_, _ = database.DB.NewInsert().Model(&domain.Role{
		Role:        "user",
		Description: "Basic User",
		CreatedAt:   time.Now(),
	}).Ignore().Exec(ctx)

	authRepo := auth.NewRepository(database.DB)
	err = authRepo.CreateUser(ctx, user)
	require.NoError(t, err)

	token := loginAs(t, "limited@test.com", "Test@1234")
	require.NotEmpty(t, token, "limited user login should succeed")
	return token
}

// getAdminID fetches the first user's ID using an admin token.
func getAdminID(t *testing.T, adminToken string) string {
	t.Helper()
	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  adminToken,
	})
	AssertOK(t, resp)
	data := AssertDataList(t, resp)
	require.GreaterOrEqual(t, len(data), 1)
	return data[0].(map[string]interface{})["id"].(string)
}

// ──────────────────────────────────────────
// FORBIDDEN TESTS – user WITHOUT required permissions
// ──────────────────────────────────────────

func TestListUsers_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t) // "user" role has no user.view by default

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  limitedToken,
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestCreateUser_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/users",
		Token:  limitedToken,
		Body: map[string]any{
			"email": "nope@test.com", "name": "Nope",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestUpdateUser_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)
	limitedToken := seedLimitedUser(t)
	adminToken := AdminToken(t)

	adminID := getAdminID(t, adminToken)

	resp := Do(t, TestApp, Request{
		Method: "PATCH",
		Path:   "/api/v1/users/" + adminID,
		Token:  limitedToken,
		Body:   map[string]any{"name": "Hacked Name"},
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestDeleteUser_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)
	limitedToken := seedLimitedUser(t)
	adminToken := AdminToken(t)

	adminID := getAdminID(t, adminToken)

	resp := Do(t, TestApp, Request{
		Method: "DELETE",
		Path:   "/api/v1/users/" + adminID,
		Token:  limitedToken,
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestViewPermissionMatrix_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/permissions/matrix",
		Token:  limitedToken,
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestViewRoles_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles",
		Token:  limitedToken,
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestViewRolePermissions_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles/user/permissions",
		Token:  limitedToken,
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestUpdateRolePermissions_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/roles/user/permissions",
		Token:  limitedToken,
		Body:   map[string]interface{}{"permissions": []string{"user.view"}},
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestViewAuditLogs_Forbidden(t *testing.T) {
	TruncateTables(t, "users", "permission_changes_log", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/audit-logs/permission-changes",
		Token:  limitedToken,
	})
	AssertForbidden(t, resp)
	AssertMessageEnvelope(t, resp)
}

// ──────────────────────────────────────────
// AUTHORIZED TESTS – user WITH seeded role permissions
// ──────────────────────────────────────────

func TestListUsers_Authorized(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)
	limitedToken := seedLimitedUser(t) // RolePermissionsSeeder runs during seedAdminUser, so 'user' has 'user.view'

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  limitedToken,
	})
	AssertOK(t, resp)
	AssertDataList(t, resp)
}

func TestUpdateUser_Authorized(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)
	limitedToken := seedLimitedUser(t)

	authRepo := auth.NewRepository(database.DB)
	err := authRepo.GrantRolePermission(t.Context(), "user", "user.edit")
	require.NoError(t, err)

	adminToken := AdminToken(t)
	adminID := getAdminID(t, adminToken)

	resp := Do(t, TestApp, Request{
		Method: "PATCH",
		Path:   "/api/v1/users/" + adminID,
		Token:  limitedToken,
		Body:   map[string]any{"name": "Limited Updated"},
	})
	AssertOK(t, resp)
	userData := AssertDataEnvelope(t, resp)
	assert.Equal(t, "Limited Updated", userData["name"])
}

// ──────────────────────────────────────────
// PERMISSION LIST ENDPOINT (no specific permission required)
// ──────────────────────────────────────────

func TestPermissionList_AccessibleByAnyAuthUser(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	limitedToken := seedLimitedUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/permissions/list",
		Token:  limitedToken,
	})
	AssertOK(t, resp)
}

// ──────────────────────────────────────────
// SUPER ADMIN BYPASS — super_admin never gets 403
// ──────────────────────────────────────────

func TestSuperAdminBypass_AllEndpoints(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	adminToken := AdminToken(t)

	t.Run("list users", func(t *testing.T) {
		resp := Do(t, TestApp, Request{
			Method: "GET", Path: "/api/v1/users", Token: adminToken,
		})
		AssertOK(t, resp)
	})

	t.Run("create user", func(t *testing.T) {
		resp := Do(t, TestApp, Request{
			Method: "POST", Path: "/api/v1/users", Token: adminToken,
			Body: map[string]any{
				"email": "sa-test@test.com", "name": "SA Test",
				"password": "Test@1234", "roles": []string{"user"},
			},
		})
		AssertCreated(t, resp)
	})

	t.Run("view permission matrix", func(t *testing.T) {
		resp := Do(t, TestApp, Request{
			Method: "GET", Path: "/api/v1/permissions/matrix", Token: adminToken,
		})
		AssertOK(t, resp)
	})

	t.Run("view roles", func(t *testing.T) {
		resp := Do(t, TestApp, Request{
			Method: "GET", Path: "/api/v1/roles", Token: adminToken,
		})
		AssertOK(t, resp)
	})

	t.Run("view audit logs", func(t *testing.T) {
		resp := Do(t, TestApp, Request{
			Method: "GET", Path: "/api/v1/audit-logs/permission-changes", Token: adminToken,
		})
		AssertOK(t, resp)
		AssertPaginationMeta(t, resp)
	})

	t.Run("update role permissions", func(t *testing.T) {
		resp := Do(t, TestApp, Request{
			Method: "PUT", Path: "/api/v1/roles/user/permissions", Token: adminToken,
			Body: map[string]any{"permissions": []string{"user.view"}},
		})
		AssertOK(t, resp)
	})
}
