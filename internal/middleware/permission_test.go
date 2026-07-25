package middleware

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"golang-base/internal/domain"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRedisClient mocks Redis client
type MockRedisClient struct {
	mock.Mock
	data map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if val, ok := m.data[key]; ok {
		cmd := redis.NewStringCmd(ctx)
		cmd.SetVal(val)
		return cmd
	}
	cmd := redis.NewStringCmd(ctx)
	cmd.SetErr(redis.Nil)
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration interface{}) *redis.StatusCmd {
	data, _ := json.Marshal(value)
	m.data[key] = string(data)
	cmd := redis.NewStatusCmd(ctx)
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	for _, key := range keys {
		delete(m.data, key)
	}
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(int64(len(keys)))
	return cmd
}

func (m *MockRedisClient) SetData(key, value string) {
	m.data[key] = value
}

// Test repository functions
var (
	testUsers = map[string]*domain.User{
		"user-1": {ID: "user-1", Email: "admin@test.com", Roles: []string{domain.RoleSuperAdmin}, IsActive: true},
		"user-2": {ID: "user-2", Email: "user@test.com", Roles: []string{"user"}, IsActive: true},
		"user-3": {ID: "user-3", Email: "inactive@test.com", Roles: []string{"user"}, IsActive: false},
		"user-4": {ID: "user-4", Email: "guest@test.com", Roles: []string{domain.RoleAdmin}, IsActive: true},
	}

	testUserPermissions = map[string][]*domain.UserPermission{
		"user-1": {},
		"user-2": {{UserID: "user-2", Permission: "user.view"}},
		"user-4": {{UserID: "user-4", Permission: "master.edit"}},
	}

	testRolePermissions = map[string][]*domain.RolePermission{
		domain.RoleSuperAdmin: {
			{Role: domain.RoleSuperAdmin, Permission: "user.view"},
			{Role: domain.RoleSuperAdmin, Permission: "user.edit"},
		},
		"user": {
			{Role: "user", Permission: "client.view"},
		},
		domain.RoleAdmin: {
			{Role: domain.RoleAdmin, Permission: "master.view"},
		},
	}
)

// Test implementations
func testGetUserByID(ctx context.Context, id string) (*domain.User, error) {
	if user, ok := testUsers[id]; ok {
		return user, nil
	}
	return nil, assert.AnError
}

func testGetUserPermissions(ctx context.Context, userID string) ([]*domain.UserPermission, error) {
	if perms, ok := testUserPermissions[userID]; ok {
		return perms, nil
	}
	return []*domain.UserPermission{}, nil
}

func testGetRolePermissions(ctx context.Context, role string) ([]*domain.RolePermission, error) {
	if perms, ok := testRolePermissions[role]; ok {
		return perms, nil
	}
	return []*domain.RolePermission{}, nil
}

func TestCachedAuthRepository_GetUserByID(t *testing.T) {
	repo := NewCachedAuthRepository(testGetUserByID, testGetUserPermissions, testGetRolePermissions, nil)

	user, err := repo.GetUserByID(context.Background(), "user-1")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "admin@test.com", user.Email)
	assert.Equal(t, []string{domain.RoleSuperAdmin}, user.Roles)
}

func TestCachedAuthRepository_GetUserByID_NotFound(t *testing.T) {
	repo := NewCachedAuthRepository(testGetUserByID, testGetUserPermissions, testGetRolePermissions, nil)

	user, err := repo.GetUserByID(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestMatchPermission_Exact(t *testing.T) {
	assert.True(t, matchPermission("master.view", []string{"master.view"}))
	assert.True(t, matchPermission("master.view", []string{"master.edit", "master.view"}))
	assert.False(t, matchPermission("master.view", []string{"master.edit"}))
}

func TestMatchPermission_Wildcard(t *testing.T) {
	assert.True(t, matchPermission("master.view", []string{"master.*"}))
	assert.True(t, matchPermission("master.edit", []string{"master.*"}))
	assert.True(t, matchPermission("master.view", []string{"pricing.*", "master.*"}))
	assert.False(t, matchPermission("user.view", []string{"master.*"}))
	assert.False(t, matchPermission("master", []string{"master.*"})) // no dot suffix
}

func TestAllowedPermissions_SuperAdminBypass(t *testing.T) {
	// Initialize global repo
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user-1") // super_admin
		return c.Next()
	})
	app.Use(AllowedPermissions("master.edit"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAllowedPermissions_HasRolePermission(t *testing.T) {
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user-4") // guest with master.view via role
		return c.Next()
	})
	app.Use(AllowedPermissions("master.view"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAllowedPermissions_HasUserOverride(t *testing.T) {
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user-4") // guest with master.edit via user override
		return c.Next()
	})
	app.Use(AllowedPermissions("master.edit"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAllowedPermissions_Wildcard(t *testing.T) {
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user-4") // guest with master.view (role) + master.edit (override)
		return c.Next()
	})
	app.Use(AllowedPermissions("master.*"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAllowedPermissions_NoPermission(t *testing.T) {
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user-2") // user, no master permissions
		return c.Next()
	})
	app.Use(AllowedPermissions("master.*"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestAllowedPermissions_MultiplePatterns(t *testing.T) {
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user-2") // user with client.view
		return c.Next()
	})
	app.Use(AllowedPermissions("master.*", "client.view"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAllowedPermissions_Unauthenticated(t *testing.T) {
	InitPermissionMiddleware(testGetUserByID, testGetUserPermissions, testGetRolePermissions)
	defer func() { cachedRepo = nil }()

	app := fiber.New()
	// No userID set
	app.Use(AllowedPermissions("master.view"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestCachedGetRolePermissions_WithMockRedis(t *testing.T) {
	mockRedis := NewMockRedisClient()

	// Pre-populate cache
	cachedPerms := []*domain.RolePermission{{Role: "user", Permission: "client.view"}}
	data, _ := json.Marshal(cachedPerms)
	mockRedis.SetData(domain.CacheRolePermissionsKey+"user", string(data))

	repo := NewCachedAuthRepository(testGetUserByID, testGetUserPermissions, testGetRolePermissions, nil)

	ctx := context.Background()
	perms, err := repo.CachedGetRolePermissions(ctx, "user")

	assert.NoError(t, err)
	assert.NotNil(t, perms)
}
