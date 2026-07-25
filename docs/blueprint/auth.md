# Auth Module Blueprint

## Data Models

### role_permissions

| Field      | Type         | Constraints   | Description                          |
| ---------- | ------------ | ------------- | ------------------------------------ |
| role       | varchar(50)  | PK            | Role name                            |
| permission | varchar(100) | PK            | Permission key (e.g., "user.create") |
| created_at | timestamp    | DEFAULT NOW() | Creation timestamp                   |

> **Presence = allowed**. No row = permission not granted.

### user_permissions (optional granular overrides)

| Field      | Type         | Constraints      | Description        |
| ---------- | ------------ | ---------------- | ------------------ |
| user_id    | bigint       | FK(users.id), PK | User reference     |
| permission | varchar(100) | PK               | Permission key     |
| created_at | timestamp    | DEFAULT NOW()    | Creation timestamp |

> **Presence = allowed**. No row = falls back to role permissions.

## Roles

| Role        | Code        | Description                                                  |
| ----------- | ----------- | ------------------------------------------------------------ |
| Super Admin | super_admin | Full system access, all permissions (not shown in UI matrix) |
| Admin       | admin       | System administration and management                         |
| User        | user        | Standard user access                                         |

## Permission Matrix

> **Note**: Super Admin row is hidden in UI but has ALL permissions by default.

| Permission  | Description         | admin | user |
| ----------- | ------------------- | :---: | :--: |
| **user**    | _(User management)_ |       |      |
| user.create | Buat user baru      |   ✓   |  -   |
| user.edit   | Edit data user      |   ✓   |  -   |
| user.view   | Lihat data user     |   ✓   |  ✓   |
| user.delete | Hapus user          |   ✓   |  -   |
| **role**    | _(Role management)_ |       |      |
| role.view   | Lihat data role     |   ✓   |  -   |
| role.edit   | Edit data role      |   ✓   |  -   |

## Audit Logging

### permission_changes_log

| Field          | Type         | Constraints            | Description                          |
| -------------- | ------------ | ---------------------- | ------------------------------------ |
| id             | uuid         | PK                     | Record identifier                    |
| action         | varchar(20)  | NOT NULL               | grant (INSERT) or revoke (DELETE)    |
| target_type    | varchar(20)  | NOT NULL               | Target type: role, user_permission   |
| target_role    | varchar(50)  | NULLABLE               | Role name (if action on role)        |
| target_user_id | bigint       | NULLABLE               | User ID (if action on user override) |
| permission     | varchar(100) | NOT NULL               | Permission key affected              |
| changed_by     | bigint       | FK(users.id), NOT NULL | Who made the change                  |
| reason         | varchar(255) | NULLABLE               | Optional reason/notes                |
| ip_address     | varchar(45)  | NULLABLE               | Client IP address                    |
| user_agent     | varchar(500) | NULLABLE               | Client user agent                    |
| created_at     | timestamp    | DEFAULT NOW()          | When change occurred                 |
