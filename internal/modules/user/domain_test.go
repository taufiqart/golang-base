package user

import (
	"encoding/json"
	"testing"
	"time"

	"golang-base/internal/domain"

	"github.com/stretchr/testify/assert"
)

// Test User Entity

func TestUser_JSONSerialization(t *testing.T) {
	user := &domain.User{
		ID:        "test-id-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Roles:     []string{"manager"},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded domain.User
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, user.Email, decoded.Email)
	assert.Equal(t, user.Name, decoded.Name)
}

func TestUser_PasswordExcluded(t *testing.T) {
	password := "secret123"
	user := &domain.User{
		ID:       "test-id",
		Email:    "test@example.com",
		Password: &password,
		Name:     "Test User",
		Roles:    []string{"manager"},
		IsActive: true,
	}

	data, err := json.Marshal(user)
	assert.NoError(t, err)

	// Password should not be in JSON (json:"-" tag)
	assert.NotContains(t, string(data), "secret123")
}

func TestUser_Fields(t *testing.T) {
	user := &domain.User{
		ID:       "user-123",
		Email:    "user@example.com",
		Name:     "John Doe",
		Roles:    []string{"user"},
		IsActive: true,
	}

	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, []string{"user"}, user.Roles)
}

// Test Mock Implementation

func TestMockUserRepository_GetByID(t *testing.T) {
	repo := &mockUserRepository{
		users: map[string]*domain.User{
			"1": {ID: "1", Email: "user1@test.com", Name: "User 1", Roles: []string{"user"}},
			"2": {ID: "2", Email: "user2@test.com", Name: "User 2", Roles: []string{"user"}},
		},
	}

	user, err := repo.GetByID("1")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user1@test.com", user.Email)

	_, err = repo.GetByID("999")
	assert.Error(t, err)
}

func TestMockUserRepository_GetByEmail(t *testing.T) {
	repo := &mockUserRepository{
		users: map[string]*domain.User{
			"1": {ID: "1", Email: "user1@test.com", Name: "User 1", Roles: []string{"user"}},
		},
	}

	user, err := repo.GetByEmail("user1@test.com")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "1", user.ID)

	_, err = repo.GetByEmail("nonexistent@test.com")
	assert.Error(t, err)
}

func TestMockUserRepository_Create(t *testing.T) {
	repo := &mockUserRepository{
		users: map[string]*domain.User{},
	}

	user := &domain.User{Email: "new@test.com", Name: "New User", Roles: []string{"user"}}
	err := repo.Create(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Len(t, repo.users, 1)
}

// Mock repository for testing
type mockUserRepository struct {
	users map[string]*domain.User
}

func (m *mockUserRepository) GetByID(id string) (*domain.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, assert.AnError
}

func (m *mockUserRepository) GetByEmail(email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockUserRepository) Create(user *domain.User) error {
	user.ID = "generated-id"
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Update(user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) List(limit, offset int) ([]*domain.User, error) {
	result := make([]*domain.User, 0)
	count := 0
	for _, user := range m.users {
		if count >= offset && count < offset+limit {
			result = append(result, user)
		}
		count++
	}
	return result, nil
}

func (m *mockUserRepository) Count() (int, error) {
	return len(m.users), nil
}
