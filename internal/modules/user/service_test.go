package user

import (
	"context"
	"testing"

	"golang-base/internal/domain"

	"github.com/stretchr/testify/assert"
)

// MockUserRepository for testing
type MockUserRepository struct {
	users map[string]*domain.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: map[string]*domain.User{},
	}
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, assert.AnError
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, assert.AnError
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = "generated-id"
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
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

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int, error) {
	return len(m.users), nil
}

// Test NewService

func TestNewService(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo)

	assert.NotNil(t, svc)
}

// Test GetProfile

func TestGetProfile_UserFound(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["1"] = &domain.User{ID: "1", Email: "test@example.com", Name: "Test User", Roles: []string{"user"}}

	svc := NewService(repo)

	user, err := svc.GetProfile(context.Background(), "1")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo)

	user, err := svc.GetProfile(context.Background(), "999")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGetProfile_MultipleUsers(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["1"] = &domain.User{ID: "1", Email: "user1@test.com", Name: "User 1", Roles: []string{"user"}}
	repo.users["2"] = &domain.User{ID: "2", Email: "user2@test.com", Name: "User 2", Roles: []string{"user"}}
	repo.users["3"] = &domain.User{ID: "3", Email: "user3@test.com", Name: "User 3", Roles: []string{"user"}}

	svc := NewService(repo)

	// Test each user
	for _, id := range []string{"1", "2", "3"} {
		user, err := svc.GetProfile(context.Background(), id)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, id, user.ID)
	}

	// Test non-existent user
	_, err := svc.GetProfile(context.Background(), "999")
	assert.Error(t, err)
}

// Test List

func TestList_Users(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["1"] = &domain.User{ID: "1", Email: "user1@test.com", Name: "User 1", Roles: []string{"user"}}
	repo.users["2"] = &domain.User{ID: "2", Email: "user2@test.com", Name: "User 2", Roles: []string{"admin"}}
	repo.users["3"] = &domain.User{ID: "3", Email: "user3@test.com", Name: "User 3", Roles: []string{"user"}}

	svc := NewService(repo)

	users, total, err := svc.List(context.Background(), 10, 0)

	assert.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Equal(t, 3, total)
}

func TestList_WithPagination(t *testing.T) {
	repo := NewMockUserRepository()
	for i := 1; i <= 5; i++ {
		repo.users[string(rune('0'+i))] = &domain.User{
			ID:    string(rune('0' + i)),
			Email: "user@test.com",
			Name:  "User",
			Roles: []string{"user"},
		}
	}

	svc := NewService(repo)

	// Get first page
	users, total, err := svc.List(context.Background(), 2, 0)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, 5, total)

	// Get second page
	users, total, err = svc.List(context.Background(), 2, 2)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, 5, total)
}

func TestList_Empty(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewService(repo)

	users, total, err := svc.List(context.Background(), 10, 0)

	assert.NoError(t, err)
	assert.Len(t, users, 0)
	assert.Equal(t, 0, total)
}
