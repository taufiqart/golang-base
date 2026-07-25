package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheConstants(t *testing.T) {
	assert.Equal(t, "user:permissions:", CacheUserPermissionsKey)
	assert.Equal(t, "role:permissions:", CacheRolePermissionsKey)
	assert.Equal(t, 24*time.Hour, CacheTTL)
}

func TestUserPermissionsCacheKey(t *testing.T) {
	userID := "test-user-123"
	expected := "user:permissions:test-user-123"
	assert.Equal(t, expected, CacheUserPermissionsKey+userID)
}

func TestRolePermissionsCacheKey(t *testing.T) {
	role := "admin"
	expected := "role:permissions:admin"
	assert.Equal(t, expected, CacheRolePermissionsKey+role)
}
