package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser_Success(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/users",
		Token:  token,
		Body:   map[string]any{"email": "new@test.com", "name": "New User", "password": "Test@1234", "roles": []string{"user"}},
	})
	AssertCreated(t, resp)

	data := AssertDataEnvelope(t, resp)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "new@test.com", data["email"])
	assert.Equal(t, "New User", data["name"])
	assert.Equal(t, []interface{}{"user"}, data["roles"])
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/users",
		Token:  token,
		Body:   map[string]any{"email": "dup@test.com", "name": "First", "password": "Test@1234", "roles": []string{"user"}},
	})

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/users",
		Token:  token,
		Body:   map[string]any{"email": "dup@test.com", "name": "Second", "password": "Test@1234", "roles": []string{"user"}},
	})
	AssertBadRequest(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestCreateUser_MissingFields(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/users",
		Token:  token,
		Body:   map[string]any{"email": ""},
	})
	AssertBadRequest(t, resp)
}

func TestListUsers(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  token,
	})
	AssertOK(t, resp)

	data := AssertDataList(t, resp)
	assert.Len(t, data, 1)
	first := data[0].(map[string]interface{})
	assert.Equal(t, "admin@test.com", first["email"])
}

func TestListUsers_Pagination(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users?page=1&limit=10",
		Token:  token,
	})
	AssertOK(t, resp)

	meta := AssertPaginationMeta(t, resp)
	assert.Equal(t, float64(1), meta["total_items"])
	assert.Equal(t, float64(1), meta["current_page"])
	assert.Equal(t, float64(10), meta["limit"])
}

func TestGetUser(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  token,
	})
	AssertOK(t, resp)
	data := AssertDataList(t, resp)
	require.Len(t, data, 1)
	adminID := data[0].(map[string]interface{})["id"].(string)

	resp = Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/" + adminID,
		Token:  token,
	})
	AssertOK(t, resp)

	user := AssertDataEnvelope(t, resp)
	assert.Equal(t, adminID, user["id"])
	assert.Equal(t, "admin@test.com", user["email"])
}

func TestGetUser_NotFound(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users/non-existent-id",
		Token:  token,
	})
	AssertNotFound(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestUpdateUser_Success(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Get admin user ID
	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  token,
	})
	AssertOK(t, resp)
	data := AssertDataList(t, resp)
	require.Len(t, data, 1)
	adminID := data[0].(map[string]interface{})["id"].(string)

	// Update user name
	resp = Do(t, TestApp, Request{
		Method: "PATCH",
		Path:   "/api/v1/users/" + adminID,
		Token:  token,
		Body:   map[string]any{"name": "Updated Admin"},
	})
	AssertOK(t, resp)

	userData := AssertDataEnvelope(t, resp)
	assert.Equal(t, "Updated Admin", userData["name"])
	assert.Equal(t, "admin@test.com", userData["email"])
}

func TestDeleteUser_Success(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	token := AdminToken(t)

	// Get admin user ID
	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/users",
		Token:  token,
	})
	AssertOK(t, resp)
	data := AssertDataList(t, resp)
	require.Len(t, data, 1)
	adminID := data[0].(map[string]interface{})["id"].(string)

	// Delete user
	resp = Do(t, TestApp, Request{
		Method: "DELETE",
		Path:   "/api/v1/users/" + adminID,
		Token:  token,
	})
	AssertOK(t, resp)
}
