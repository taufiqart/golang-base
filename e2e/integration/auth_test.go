package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_Success(t *testing.T) {
	TruncateTables(t, "users")
	seedAdminUser(t)

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body:   map[string]any{"email": "admin@test.com", "password": "Test@1234"},
	})
	AssertOK(t, resp)
	AssertJSON(t, resp)

	data := AssertDataEnvelope(t, resp)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	assert.Equal(t, "Bearer", data["token_type"])

	user, ok := data["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "admin@test.com", user["email"])
}

func TestLogin_InvalidPassword(t *testing.T) {
	TruncateTables(t, "users")
	seedAdminUser(t)

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body:   map[string]any{"email": "admin@test.com", "password": "wrong-password"},
	})
	AssertUnauthorized(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestLogin_NonExistentUser(t *testing.T) {
	TruncateTables(t, "users")

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body:   map[string]any{"email": "ghost@test.com", "password": "Test@1234"},
	})
	AssertUnauthorized(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestLogin_MissingFields(t *testing.T) {
	TruncateTables(t, "users")

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body:   map[string]any{"email": ""},
	})
	AssertBadRequest(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestMe(t *testing.T) {
	TruncateTables(t, "users")
	seedAdminUser(t)

	token := AdminToken(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/auth/me",
		Token:  token,
	})
	AssertOK(t, resp)

	data := AssertDataEnvelope(t, resp)
	assert.Equal(t, "admin@test.com", data["email"])
	assert.Equal(t, "Super Admin", data["name"])
	assert.Equal(t, []interface{}{"super_admin"}, data["roles"])
}

func TestMe_Unauthorized(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/auth/me",
	})
	AssertUnauthorized(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestMe_InvalidToken(t *testing.T) {
	TruncateTables(t, "users", "role_permissions")
	seedAdminUser(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/auth/me",
		Token:  "invalid-token-format",
	})
	AssertUnauthorized(t, resp)
	AssertMessageEnvelope(t, resp)
}

func TestRefreshToken(t *testing.T) {
	TruncateTables(t, "users")
	seedAdminUser(t)

	loginResp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body:   map[string]any{"email": "admin@test.com", "password": "Test@1234"},
	})
	loginData := AssertDataEnvelope(t, loginResp)
	refreshToken := loginData["refresh_token"].(string)

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/refresh",
		Body:   map[string]any{"refresh_token": refreshToken},
	})
	AssertOK(t, resp)

	data := AssertDataEnvelope(t, resp)
	assert.NotEmpty(t, data["access_token"])
	assert.Equal(t, "Bearer", data["token_type"])
}

func TestRefreshToken_Invalid(t *testing.T) {
	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/refresh",
		Body:   map[string]any{"refresh_token": "invalid-token"},
	})
	AssertUnauthorized(t, resp)
	AssertMessageEnvelope(t, resp)
}
