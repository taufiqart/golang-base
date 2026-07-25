-- Migration: Drop role_permissions table
-- Created: 2026-05-21

DROP TABLE IF EXISTS permission_changes_log;

DROP TABLE IF EXISTS user_permissions;

DROP TABLE IF EXISTS role_permissions;

DROP TABLE IF EXISTS user_roles;

DROP TABLE IF EXISTS roles;