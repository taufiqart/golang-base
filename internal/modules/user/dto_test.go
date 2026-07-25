package user

import (
	"encoding/json"
	"testing"
	"time"

	"golang-base/internal/domain"

	"github.com/stretchr/testify/assert"
)

// Test CreateUserRequest

func TestCreateUserRequest_Fields(t *testing.T) {
	req := &CreateUserRequest{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Roles:    []string{"user"},
	}

	assert.Equal(t, "test@example.com", req.Email)
	assert.Equal(t, "Test User", req.Name)
	assert.Equal(t, "password123", req.Password)
	assert.Equal(t, []string{"user"}, req.Roles)
}

func TestCreateUserRequest_JSON(t *testing.T) {
	req := CreateUserRequest{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Roles:    []string{"user"},
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"email":"test@example.com"`)
	assert.Contains(t, string(data), `"name":"Test User"`)
}

func TestCreateUserRequest_JSONWithPassword(t *testing.T) {
	req := CreateUserRequest{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "secret123",
		Roles:    []string{"admin"},
	}

	data, _ := json.Marshal(req)
	// Password should be in JSON for CreateUserRequest
	assert.Contains(t, string(data), `"password":"secret123"`)
}

// Test UpdateUserRequest

func TestUpdateUserRequest_Fields(t *testing.T) {
	req := &UpdateUserRequest{
		Email: "updated@example.com",
		Name:  "Updated Name",
		Roles: []string{"admin"},
	}

	assert.Equal(t, "updated@example.com", req.Email)
	assert.Equal(t, "Updated Name", req.Name)
	assert.Equal(t, []string{"admin"}, req.Roles)
}

func TestUpdateUserRequest_Partial(t *testing.T) {
	// Email only
	req := &UpdateUserRequest{
		Email: "only-email@example.com",
	}

	assert.Equal(t, "only-email@example.com", req.Email)
	assert.Empty(t, req.Name)
	assert.Empty(t, req.Roles)
}

// Test UserResponse

func TestUserResponse_Fields(t *testing.T) {
	resp := UserResponse{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"admin"},
		CreatedAt: "2024-01-01 12:00:00",
	}

	assert.Equal(t, "user-123", resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, []string{"admin"}, resp.Roles)
}

func TestUserResponse_JSONSerialization(t *testing.T) {
	resp := UserResponse{
		ID:        "user-1",
		Email:     "user@test.com",
		Name:      "User Name",
		Roles:     []string{"user"},
		CreatedAt: "2024-01-01 12:00:00",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded UserResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.ID, decoded.ID)
	assert.Equal(t, resp.Email, decoded.Email)
}

// Test ToResponse Helper

func TestToResponse(t *testing.T) {
	user := &domain.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"admin"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := ToResponse(user)

	assert.Equal(t, "user-123", resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, []string{"admin"}, resp.Roles)
	assert.NotEmpty(t, resp.CreatedAt)
}

func TestToResponse_PreservesFields(t *testing.T) {
	user := &domain.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"user"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := ToResponse(user)

	assert.Equal(t, "user-123", resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, []string{"user"}, resp.Roles)
	assert.NotEmpty(t, resp.CreatedAt)
}

// Test Response Mapping

func TestUserToResponse_MultipleFields(t *testing.T) {
	testCases := []struct {
		name   string
		user   *domain.User
		expect func(*testing.T, UserResponse)
	}{
		{
			name: "Admin User",
			user: &domain.User{ID: "1", Email: "admin@test.com", Name: "Admin", Roles: []string{"admin"}},
			expect: func(t *testing.T, resp UserResponse) {
				assert.Equal(t, "1", resp.ID)
				assert.Equal(t, "admin@test.com", resp.Email)
			},
		},
		{
			name: "Regular User",
			user: &domain.User{ID: "2", Email: "user@test.com", Name: "User", Roles: []string{"user"}},
			expect: func(t *testing.T, resp UserResponse) {
				assert.Equal(t, "2", resp.ID)
				assert.Equal(t, "user@test.com", resp.Email)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := ToResponse(tc.user)
			tc.expect(t, resp)
		})
	}
}
