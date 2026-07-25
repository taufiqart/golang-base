package auth

import (
	"encoding/json"
	"testing"
	"time"

	"golang-base/internal/domain"

	"github.com/stretchr/testify/assert"
)

// Test DTOs

func TestRegisterRequest_Validation(t *testing.T) {
	req := &RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Roles:    []string{"user"},
	}

	assert.Equal(t, "test@example.com", req.Email)
	assert.Equal(t, "password123", req.Password)
	assert.Equal(t, "Test User", req.Name)
	assert.Equal(t, []string{"user"}, req.Roles)
}

func TestLoginRequest_Fields(t *testing.T) {
	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	assert.Equal(t, "test@example.com", req.Email)
	assert.Equal(t, "password123", req.Password)
}

func TestRefreshTokenRequest_Fields(t *testing.T) {
	req := &RefreshTokenRequest{
		RefreshToken: "some-jwt-token",
	}

	assert.Equal(t, "some-jwt-token", req.RefreshToken)
}

func TestAuthResponse_Success(t *testing.T) {
	resp := &AuthResponse{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		User: &UserResponse{
			ID:    "user-123",
			Email: "test@example.com",
			Name:  "Test User",
			Roles: []string{"user"},
		},
	}

	assert.Equal(t, "access-token-123", resp.AccessToken)
	assert.Equal(t, "refresh-token-456", resp.RefreshToken)
	assert.Equal(t, "test@example.com", resp.User.Email)
}

func TestUserResponse_Serialization(t *testing.T) {
	resp := UserResponse{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"admin"},
		IsActive:  true,
		CreatedAt: "2024-01-01 12:00:00",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded UserResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.Email, decoded.Email)
	assert.Equal(t, resp.Roles, decoded.Roles)
}

func TestUserResponse_PasswordExcluded(t *testing.T) {
	resp := UserResponse{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Roles: []string{"user"},
	}

	data, _ := json.Marshal(resp)

	// Password should not be in JSON (it's not part of UserResponse)
	assert.NotContains(t, string(data), "password")
}

func TestCreateUserRequest_Fields(t *testing.T) {
	req := &CreateUserRequest{
		Email:    "new@example.com",
		Password: "newpassword",
		Name:     "New User",
		Roles:    []string{"user"},
	}

	assert.Equal(t, "new@example.com", req.Email)
	assert.Equal(t, "newpassword", req.Password)
	assert.Equal(t, "New User", req.Name)
}

func TestUpdateUserRequest_Fields(t *testing.T) {
	isActive := true
	req := &UpdateUserRequest{
		Name:     "Updated Name",
		Roles:    []string{"admin"},
		IsActive: &isActive,
	}

	assert.Equal(t, "Updated Name", req.Name)
	assert.Equal(t, []string{"admin"}, req.Roles)
	assert.NotNil(t, req.IsActive)
	assert.True(t, *req.IsActive)
}

func TestRoleRequest_Fields(t *testing.T) {
	req := &RoleRequest{
		Role:        "new_role",
		Description: "Description here",
	}

	assert.Equal(t, "new_role", req.Role)
	assert.Equal(t, "Description here", req.Description)
}

func TestRoleResponse_WithDescription(t *testing.T) {
	resp := &RoleResponse{
		Role:        "admin",
		Description: "Manager Role",
		CreatedAt:   "2024-01-01 12:00:00",
	}

	assert.Equal(t, "admin", resp.Role)
	assert.Equal(t, "Manager Role", resp.Description)
}

func TestUpdatePermissionsRequest_Fields(t *testing.T) {
	req := &UpdatePermissionsRequest{
		Permissions: []string{"user.view", "user.edit", "client.view"},
	}

	assert.Len(t, req.Permissions, 3)
	assert.Contains(t, req.Permissions, "user.view")
}

func TestGrantPermissionRequest_Fields(t *testing.T) {
	req := &GrantPermissionRequest{
		Permission: "user.delete",
	}

	assert.Equal(t, "user.delete", req.Permission)
}

func TestAuditLogQuery_Defaults(t *testing.T) {
	filter := &AuditLogQuery{}

	assert.Equal(t, 0, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestAuditLogQuery_WithFilters(t *testing.T) {
	action := "grant"

	filter := &AuditLogQuery{
		Action: &action,
		Limit:  25,
	}

	assert.Equal(t, "grant", *filter.Action)
	assert.Equal(t, 25, filter.Limit)
}

func TestPermissionMatrixResponse_Matrix(t *testing.T) {
	resp := PermissionMatrixResponse{
		Permissions: []string{"user.view", "user.edit"},
		Roles:       []string{"admin", "user"},
		Matrix: map[string]map[string]bool{
			"admin": {"user.view": true, "user.edit": true},
			"user":  {"user.view": true, "user.edit": false},
		},
	}

	assert.Len(t, resp.Permissions, 2)
	assert.Len(t, resp.Roles, 2)
	assert.True(t, resp.Matrix["admin"]["user.view"])
	assert.True(t, resp.Matrix["user"]["user.view"])
	assert.False(t, resp.Matrix["user"]["user.edit"])
}

// Test Helper Functions

func TestToUserResponse(t *testing.T) {
	user := &domain.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"user"},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := ToUserResponse(user)

	assert.Equal(t, "user-123", resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, []string{"user"}, resp.Roles)
	assert.True(t, resp.IsActive)
}

func TestToRoleResponse(t *testing.T) {
	role := &domain.Role{
		Role:        "admin",
		Description: "Manager",
		CreatedAt:   time.Now(),
	}

	resp := ToRoleResponse(role)

	assert.Equal(t, "admin", resp.Role)
	assert.Equal(t, "Manager", resp.Description)
}

// Note: ToPermissionMatrixResponse is removed because /permissions/matrix
// now reads from database. Use service.GetPermissionMatrix() instead.
