package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionMatrix(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/permissions/matrix",
		Token:  token,
	})
	AssertOK(t, resp)

	matrix := AssertDataEnvelope(t, resp)
	require.NotNil(t, matrix)

	// Check required fields
	for _, field := range []string{"permissions", "roles", "matrix"} {
		val, ok := matrix[field]
		assert.True(t, ok, "field '%s' should exist", field)
		assert.NotNil(t, val, "field '%s' should not be nil", field)
	}

	// Validate types
	perms, ok := matrix["permissions"].([]interface{})
	assert.True(t, ok, "permissions should be an array")
	assert.Greater(t, len(perms), 0, "should have at least 1 permission")

	roles, ok := matrix["roles"].([]interface{})
	assert.True(t, ok, "roles should be an array")
	assert.Greater(t, len(roles), 0, "should have at least 1 role")

	permMatrix, ok := matrix["matrix"].(map[string]interface{})
	assert.True(t, ok, "matrix should be an object")
	_ = permMatrix
}

func TestPermissionList(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/permissions/list",
		Token:  token,
	})
	AssertOK(t, resp)

	list := AssertDataEnvelope(t, resp)
	require.NotNil(t, list)

	// Check permissions array
	permDefs, ok := list["permissions"].([]interface{})
	assert.True(t, ok, "permissions should be an array")
	if len(permDefs) > 0 {
		perm, ok := permDefs[0].(map[string]interface{})
		assert.True(t, ok, "permission entry should be an object")
		for _, field := range []string{"key", "description", "category"} {
			assert.Contains(t, perm, field, "permission should have '%s'", field)
		}
	}

	// Check roles array
	roles, ok := list["roles"].([]interface{})
	assert.True(t, ok, "roles should be an array")
	assert.Greater(t, len(roles), 0, "should have at least 1 role")
}

func TestRolesList(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles",
		Token:  token,
	})
	AssertOK(t, resp)

	roles := AssertDataList(t, resp)
	require.GreaterOrEqual(t, len(roles), 1)

	role := roles[0].(map[string]interface{})
	assert.Contains(t, role, "role")
	assert.Contains(t, role, "permissions")

	// Verify expected roles are present
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.(map[string]interface{})["role"].(string)
	}

	expectedRoles := []string{"super_admin", "admin", "user"}
	for _, expected := range expectedRoles {
		assert.Contains(t, roleNames, expected, "role '%s' should exist", expected)
	}
}

func TestGetRolePermissions(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles/user/permissions",
		Token:  token,
	})
	AssertOK(t, resp)

	data := AssertDataEnvelope(t, resp)
	require.NotNil(t, data)
	perms, ok := data["permissions"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, perms, "user.view")
}

func TestGetRolePermissions_NotFound(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles/nonexistent_role/permissions",
		Token:  token,
	})
	AssertNotFound(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestUpdateRolePermissions(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/roles/user/permissions",
		Token:  token,
		Body: map[string]any{
			"permissions": []string{"user.view"},
		},
	})
	AssertOK(t, resp)

	// Verify by fetching updated permissions
	resp = Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles/user/permissions",
		Token:  token,
	})
	AssertOK(t, resp)

	data := AssertDataEnvelope(t, resp)
	perms, ok := data["permissions"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, perms, "user.view")
}

func TestGetUserPermissions(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Perm Test", "email": "perm@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/" + userID + "/permissions",
		Token:  token,
	})
	AssertOK(t, resp)

	data := AssertDataEnvelope(t, resp)
	assert.Contains(t, data, "permissions")
}

func TestGrantUserPermission(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Set Perm Test", "email": "setperm@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	resp := Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
		Body:   map[string]interface{}{"granted": true, "reason": "E2E test"},
	})
	AssertOK(t, resp)

	// Verify permission was granted (response is map[string]bool, not array)
	getResp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/" + userID + "/permissions",
		Token:  token,
	})
	AssertOK(t, getResp)
	data := AssertDataEnvelope(t, getResp)
	perms, ok := data["permissions"].(map[string]interface{})
	assert.True(t, ok, "permissions should be a map[string]bool")
	assert.Equal(t, true, perms["master.view"], "master.view should be granted")
}

func TestRevokeUserPermission(t *testing.T) {
	TruncateTables(t, "users", "user_permissions", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Remove Perm Test", "email": "rmperm@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	// First grant a permission
	Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
		Body:   map[string]interface{}{"granted": true, "reason": "E2E test"},
	})

	// Then revoke it
	resp := Do(t, TestApp, Request{
		Method: "DELETE",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
	})
	AssertOK(t, resp)

	// Verify permission was revoked (response is map[string]bool, not array)
	getResp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/" + userID + "/permissions",
		Token:  token,
	})
	AssertOK(t, getResp)
	data := AssertDataEnvelope(t, getResp)
	perms, ok := data["permissions"].(map[string]interface{})
	assert.True(t, ok, "permissions should be a map[string]bool")
	assert.NotContains(t, perms, "master.view", "master.view should be revoked")
}

func TestRevokeUserPermissionOverride(t *testing.T) {
	TruncateTables(t, "users", "user_permissions", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Revoke Perm Test", "email": "revoke-override@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	// Revoke a permission explicitly (override with granted: false)
	resp := Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
		Body:   map[string]interface{}{"granted": false, "reason": "E2E revoke override test"},
	})
	AssertOK(t, resp)

	// Verify permission is revoked (false in map)
	getResp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/" + userID + "/permissions",
		Token:  token,
	})
	AssertOK(t, getResp)
	data := AssertDataEnvelope(t, getResp)
	perms, ok := data["permissions"].(map[string]interface{})
	assert.True(t, ok, "permissions should be a map[string]bool")
	assert.Equal(t, false, perms["master.view"], "master.view should be revoked (false)")
}

func TestExpiredUserPermission(t *testing.T) {
	TruncateTables(t, "users", "user_permissions", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Expired Perm Test", "email": "expired@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	// Grant a permission with past expiration date
	resp := Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
		Body:   map[string]interface{}{"granted": true, "expires_at": "2020-01-01T00:00:00Z", "reason": "E2E expired test"},
	})
	AssertOK(t, resp)

	// Verify expired permission is excluded from GetUserPermissions
	getResp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/" + userID + "/permissions",
		Token:  token,
	})
	AssertOK(t, getResp)
	data := AssertDataEnvelope(t, getResp)
	perms, ok := data["permissions"].(map[string]interface{})
	assert.True(t, ok, "permissions should be a map[string]bool")
	assert.NotContains(t, perms, "master.view", "master.view should be expired and excluded")
}

func TestAuditLogs(t *testing.T) {
	TruncateTables(t, "users", "user_permissions", "role_permissions", "permission_changes_log")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user to grant permissions to (creates audit log entries)
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Audit Test", "email": "audit@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	// Grant permission to user (creates audit log entry)
	Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
	})

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/audit-logs/permission-changes",
		Token:  token,
	})
	AssertOK(t, resp)

	logs := AssertDataList(t, resp)
	if len(logs) > 0 {
		logEntry := logs[0].(map[string]interface{})
		for _, field := range []string{"id", "action", "permission", "created_at"} {
			assert.Contains(t, logEntry, field, "log entry should have '%s'", field)
		}
	}

	AssertPaginationMeta(t, resp)
}

func TestAuditLogs_Filter(t *testing.T) {
	TruncateTables(t, "users", "user_permissions", "role_permissions", "permission_changes_log")
	seedAdminUser(t)

	token := AdminToken(t)

	// Create a user to grant permissions to (creates audit log entries)
	createResp := Do(t, TestApp, Request{
		Method: "POST", Path: "/api/v1/users", Token: token,
		Body: map[string]any{
			"name": "Audit Filter Test", "email": "auditfilter@test.com",
			"password": "Test@1234", "roles": []string{"user"},
		},
	})
	AssertCreated(t, createResp)
	created := AssertDataEnvelope(t, createResp)
	userID := created["id"].(string)

	// Grant permission to user (creates audit log entry)
	Do(t, TestApp, Request{
		Method: "PUT",
		Path:   "/api/v1/users/" + userID + "/permissions/master.view",
		Token:  token,
	})

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/audit-logs/permission-changes?action=grant&limit=5",
		Token:  token,
	})
	AssertOK(t, resp)

	meta := AssertPaginationMeta(t, resp)
	_ = meta
}

func TestAuditLogs_Unauthorized(t *testing.T) {
	TruncateTables(t, "users", "role_permissions", "permission_changes_log")
	seedAdminUser(t)

	_ = AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/audit-logs/permission-changes",
		// No token
	})
	AssertUnauthorized(t, resp)
}

func TestPermissionMatrix_Unauthorized(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/permissions/matrix",
		// No token
	})
	AssertUnauthorized(t, resp)
}

func TestRolesList_Unauthorized(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/roles",
		// No token
	})
	AssertUnauthorized(t, resp)
}
