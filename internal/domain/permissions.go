package domain

import (
	"sort"
	"strings"
	"sync"
)

// PermissionDefinition represents a permission with metadata
type PermissionDefinition struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// PermissionGroup represents a group of permissions
type PermissionGroup struct {
	Name        string
	Permissions []string
}

var (
	registeredPermissions = make(map[string]string)
	registryMutex         sync.RWMutex
)

func init() {
	// Core user permissions
	RegisterPermission("user.create", "Create new user")
	RegisterPermission("user.edit", "Edit user data")
	RegisterPermission("user.view", "View user data")
	RegisterPermission("user.delete", "Delete user")

	// Core role permissions
	RegisterPermission("role.view", "View role data")
	RegisterPermission("role.edit", "Edit role data")

	// Core storage permissions
	RegisterPermission("storage.upload", "Upload files to storage")
	RegisterPermission("storage.view", "View and generate presigned URLs for storage files")
	RegisterPermission("storage.delete", "Delete files from storage")
}

// RegisterPermission adds a permission to the global registry from any module.
func RegisterPermission(key, description string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	registeredPermissions[key] = description
}

// GetPermissions returns all available registered permissions.
func GetPermissions() map[string]string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	perms := make(map[string]string, len(registeredPermissions))
	for k, v := range registeredPermissions {
		perms[k] = v
	}
	return perms
}

// AllPermissions returns all permission keys, sorted by category then name.
func AllPermissions() []string {
	perms := GetPermissions()
	keys := make([]string, 0, len(perms))
	for k := range perms {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetPermissionDescription returns description for a permission.
func GetPermissionDescription(permission string) string {
	if desc, ok := GetPermissions()[permission]; ok {
		return desc
	}
	return permission
}

// GetPermissionCategory extracts category from permission key (e.g., "agent.view" -> "agent")
func GetPermissionCategory(permission string) string {
	if idx := strings.Index(permission, "."); idx > 0 {
		return permission[:idx]
	}
	return "unknown"
}

// PermissionGroups returns permissions organized by group derived from GetPermissions()
func PermissionGroups() []PermissionGroup {
	perms := GetPermissions()

	// Collect unique categories from permission keys
	catSet := make(map[string]bool)
	for perm := range perms {
		catSet[GetPermissionCategory(perm)] = true
	}

	// Build sorted category list
	categories := make([]string, 0, len(catSet))
	for cat := range catSet {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	// Group permissions by category
	groups := make([]PermissionGroup, 0, len(categories))
	for _, cat := range categories {
		var groupPerms []string
		for perm := range perms {
			if GetPermissionCategory(perm) == cat {
				groupPerms = append(groupPerms, perm)
			}
		}
		sort.Strings(groupPerms)
		groups = append(groups, PermissionGroup{
			Name:        cat,
			Permissions: groupPerms,
		})
	}

	return groups
}
