# User Module Blueprint

## Data Models

### users

| Field      | Type         | Constraints      | Description                          |
| ---------- | ------------ | ---------------- | ------------------------------------ |
| id         | uuidv7       | PK               | User unique identifier               |
| email      | varchar(255) | UNIQUE, NOT NULL | User email address                   |
| password   | text         | NOT NULL         | Hashed password                      |
| role       | varchar(50)  | NOT NULL         | User role (super_admin, admin, user) |
| is_active  | boolean      | DEFAULT true     | Account active status                |
| created_at | timestamp    | DEFAULT NOW()    | Creation timestamp                   |
| updated_at | timestamp    | DEFAULT NOW()    | Last update timestamp                |
| deleted_at | timestamp    | NULLABLE         | Soft delete timestamp                |

## Service Operations

- `GetProfile(ctx context.Context, id string) (*domain.User, error)`
- `UpdateProfile(ctx context.Context, id string, req interface{}) (*domain.User, error)`
- `CreateUser(ctx context.Context, req interface{}) (*domain.User, error)`
- `UpdateUser(ctx context.Context, id string, req interface{}) (*domain.User, error)`
- `List(ctx context.Context, limit, offset int) ([]*domain.User, int, error)`

## API Endpoints

- `GET /api/v1/users/me` - Get current user profile
- `PUT /api/v1/users/me` - Update current user profile
- `POST /api/v1/users` - Create a new user (Admin only)
- `PUT /api/v1/users/:id` - Update a user (Admin only)
- `GET /api/v1/users` - List all users (Admin only)
