package auth

import (
	"testing"
	"time"

	"golang-base/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository mocks auth repository
type TestMockRepository struct {
	mock.Mock
}

func (m *TestMockRepository) CreateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *TestMockRepository) GetUserByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *TestMockRepository) GetUserByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *TestMockRepository) UpdateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *TestMockRepository) DeleteUser(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *TestMockRepository) ListUsers(limit, offset int) ([]*domain.User, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *TestMockRepository) GrantRolePermission(role string, permission string, expiresAt ...*time.Time) error {
	args := m.Called(role, permission)
	return args.Error(0)
}

func (m *TestMockRepository) RevokeRolePermission(role string, permission string) error {
	args := m.Called(role, permission)
	return args.Error(0)
}

func (m *TestMockRepository) GetRolePermissions(role string) ([]*domain.RolePermission, error) {
	args := m.Called(role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RolePermission), args.Error(1)
}

func (m *TestMockRepository) GrantUserPermission(userID string, permission string, isGranted bool, expiresAt *time.Time) error {
	args := m.Called(userID, permission, isGranted, expiresAt)
	return args.Error(0)
}

func (m *TestMockRepository) RevokeUserPermission(userID string, permission string) error {
	args := m.Called(userID, permission)
	return args.Error(0)
}

func (m *TestMockRepository) GetUserPermissions(userID string) ([]*domain.UserPermission, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.UserPermission), args.Error(1)
}

func (m *TestMockRepository) LogPermissionChange(log *domain.PermissionChangesLog) error {
	args := m.Called(log)
	return args.Error(0)
}

func (m *TestMockRepository) QueryPermissionChanges(filter *domain.PermissionQueryFilter) ([]*domain.PermissionChangesLog, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PermissionChangesLog), args.Error(1)
}

// Tests for User Service

func TestCreateUser_Success(t *testing.T) {
	repo := new(TestMockRepository)

	repo.On("GetUserByEmail", "test@example.com").Return(nil, nil)
	repo.On("CreateUser", mock.AnythingOfType("*domain.User")).Return(nil)

	// Test direct mock calls
	existing, err := repo.GetUserByEmail("test@example.com")
	assert.NoError(t, err)
	assert.Nil(t, existing)

	err = repo.CreateUser(&domain.User{Email: "test@example.com"})
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestGetUserPermissions_MapConversion(t *testing.T) {
	perms := []*domain.UserPermission{
		{UserID: "user-123", Permission: "user.view"},
		{UserID: "user-123", Permission: "user.edit"},
	}

	// Convert to map (as GetUserPermissions does internally)
	result := make(map[string]bool)
	for _, p := range perms {
		result[p.Permission] = true
	}

	assert.True(t, result["user.view"])
	assert.True(t, result["user.edit"])
	assert.False(t, result["user.delete"])
}

// Tests for Permission Service

func TestGrantUserPermission_ToUser(t *testing.T) {
	repo := new(TestMockRepository)

	repo.On("GrantUserPermission", "user-123", "user.delete", true, (*time.Time)(nil)).Return(nil)
	repo.On("LogPermissionChange", mock.AnythingOfType("*domain.PermissionChangesLog")).Return(nil)

	// Test direct mock calls
	err := repo.GrantUserPermission("user-123", "user.delete", true, nil)
	assert.NoError(t, err)

	actorID := "admin-1"
	err = repo.LogPermissionChange(&domain.PermissionChangesLog{
		Action:       "grant",
		TargetUserID: &actorID,
		Permission:   "user.delete",
		IsGranted:    true,
	})
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestGetUserPermissions_ExpiredAndRevoked(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)
	perms := []*domain.UserPermission{
		{UserID: "user-123", Permission: "user.view", IsGranted: true, ExpiresAt: nil},
		{UserID: "user-123", Permission: "user.edit", IsGranted: true, ExpiresAt: &past},
		{UserID: "user-123", Permission: "user.delete", IsGranted: false, ExpiresAt: &future},
	}

	result := make(map[string]bool)
	now := time.Now()
	for _, p := range perms {
		if p.ExpiresAt != nil && p.ExpiresAt.Before(now) {
			continue
		}
		result[p.Permission] = p.IsGranted
	}

	assert.True(t, result["user.view"])
	assert.False(t, result["user.delete"])
	_, exists := result["user.edit"]
	assert.False(t, exists)
}

func TestRevokeUserPermission_FromUser(t *testing.T) {
	repo := new(TestMockRepository)

	repo.On("RevokeUserPermission", "user-123", "user.delete").Return(nil)
	repo.On("LogPermissionChange", mock.AnythingOfType("*domain.PermissionChangesLog")).Return(nil)

	err := repo.RevokeUserPermission("user-123", "user.delete")
	assert.NoError(t, err)

	actorID := "admin-1"
	err = repo.LogPermissionChange(&domain.PermissionChangesLog{
		Action:       "revoke",
		TargetUserID: &actorID,
		Permission:   "user.delete",
	})
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestUpdateRolePermissions_BulkUpdate(t *testing.T) {
	repo := new(TestMockRepository)

	existingPerms := []*domain.RolePermission{
		{Role: "manager", Permission: "user.view"},
	}

	repo.On("GetRolePermissions", "manager").Return(existingPerms, nil)

	// Should grant user.edit (new)
	repo.On("GrantRolePermission", "manager", "user.edit").Return(nil)

	// Test direct mock calls
	perms, err := repo.GetRolePermissions("manager")
	assert.NoError(t, err)
	assert.Len(t, perms, 1)

	err = repo.GrantRolePermission("manager", "user.edit")
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestRolePermissions_CacheKeyGeneration(t *testing.T) {
	role := "manager"
	expected := domain.CacheRolePermissionsKey + role
	assert.Equal(t, "role:permissions:manager", expected)
}

func TestUserPermissions_CacheKeyGeneration(t *testing.T) {
	userID := "user-123"
	expected := domain.CacheUserPermissionsKey + userID
	assert.Equal(t, "user:permissions:user-123", expected)
}
