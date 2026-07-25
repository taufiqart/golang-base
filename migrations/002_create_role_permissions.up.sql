-- Migration: Create role_permissions table
-- Created: 2026-05-21
-- Note: Stores which permissions each role has
-- Default permissions are seeded via: go run cmd/seed/main.go

CREATE TABLE IF NOT EXISTS roles (
    role VARCHAR(50) PRIMARY KEY,
    description VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL REFERENCES roles (role) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles (user_id);

CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles (role);

CREATE TABLE IF NOT EXISTS role_permissions (
    role VARCHAR(50) NOT NULL REFERENCES roles (role) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    PRIMARY KEY (role, permission)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions (role);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions (permission);

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    is_granted BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    PRIMARY KEY (user_id, permission)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_user_id ON user_permissions (user_id);

CREATE INDEX IF NOT EXISTS idx_user_permissions_permission ON user_permissions (permission);

CREATE TABLE IF NOT EXISTS permission_changes_log (
    id UUID PRIMARY KEY,
    action VARCHAR(20) NOT NULL,
    target_type VARCHAR(20) NOT NULL,
    target_role VARCHAR(50),
    target_user_id UUID,
    permission VARCHAR(100) NOT NULL,
    is_granted BOOLEAN NOT NULL DEFAULT true,
    changed_by UUID NOT NULL REFERENCES users (id),
    reason VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_permission_changes_target ON permission_changes_log (
    target_type,
    target_role,
    target_user_id
);

CREATE INDEX IF NOT EXISTS idx_permission_changes_changed_by ON permission_changes_log (changed_by);