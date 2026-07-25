#!/bin/bash

# Ensure module name is provided
if [ -z "$1" ]; then
    echo "Error: Module name is required."
    echo "Usage: ./make-module.sh <module_name_lowercase>"
    exit 1
fi

MODULE_LOWER=$1
# Convert first letter to uppercase for struct names
MODULE_TITLE="$(tr '[:lower:]' '[:upper:]' <<< ${MODULE_LOWER:0:1})${MODULE_LOWER:1}"
MODULE_DIR="internal/modules/$MODULE_LOWER"

# Create directory
mkdir -p "$MODULE_DIR"

# 1. Create dto.go
cat > "$MODULE_DIR/dto.go" <<EOF
package $MODULE_LOWER

import "time"

type Create${MODULE_TITLE}Request struct {
	Name string \`json:"name" validate:"required"\`
}

type Update${MODULE_TITLE}Request struct {
	Name string \`json:"name" validate:"required"\`
}

type ${MODULE_TITLE}Response struct {
	ID        string     \`json:"id"\`
	Name      string    \`json:"name"\`
	CreatedAt time.Time \`json:"created_at"\`
	UpdatedAt time.Time \`json:"updated_at"\`
}
EOF

# 2. Create repository.go
cat > "$MODULE_DIR/repository.go" <<EOF
package $MODULE_LOWER

import (
	"context"
	"github.com/uptrace/bun"
)

type Repository interface {
	Create(ctx context.Context, data interface{}) error
	GetByID(ctx context.Context, id string) (interface{}, error)
	List(ctx context.Context, limit, offset int) ([]interface{}, int, error)
	Update(ctx context.Context, id string, data interface{}) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	db bun.IDB
}

func NewRepository(db bun.IDB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, data interface{}) error {
	// e.g. _, err := r.db.NewInsert().Model(data).Exec(ctx)
	return nil
}

func (r *repository) GetByID(ctx context.Context, id string) (interface{}, error) {
	return nil, nil
}

func (r *repository) List(ctx context.Context, limit, offset int) ([]interface{}, int, error) {
	return nil, 0, nil
}

func (r *repository) Update(ctx context.Context, id string, data interface{}) error {
	return nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	return nil
}
EOF

# 3. Create service.go
cat > "$MODULE_DIR/service.go" <<EOF
package $MODULE_LOWER

import (
	"context"
)

type Service interface {
	Create(ctx context.Context, req *Create${MODULE_TITLE}Request) (*${MODULE_TITLE}Response, error)
	GetByID(ctx context.Context, id string) (*${MODULE_TITLE}Response, error)
	List(ctx context.Context, limit, offset int) ([]*${MODULE_TITLE}Response, int, error)
	Update(ctx context.Context, id string, req *Update${MODULE_TITLE}Request) (*${MODULE_TITLE}Response, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req *Create${MODULE_TITLE}Request) (*${MODULE_TITLE}Response, error) {
	return &${MODULE_TITLE}Response{}, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*${MODULE_TITLE}Response, error) {
	return &${MODULE_TITLE}Response{}, nil
}

func (s *service) List(ctx context.Context, limit, offset int) ([]*${MODULE_TITLE}Response, int, error) {
	return nil, 0, nil
}

func (s *service) Update(ctx context.Context, id string, req *Update${MODULE_TITLE}Request) (*${MODULE_TITLE}Response, error) {
	return &${MODULE_TITLE}Response{}, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	return nil
}
EOF

# 4. Create handler.go
cat > "$MODULE_DIR/handler.go" <<EOF
package $MODULE_LOWER

import (
	"github.com/gofiber/fiber/v2"
	"golang-base/internal/pkg/response"
	"golang-base/internal/pkg/validator"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	routes := r.Group("/$MODULE_LOWER")
	routes.Post("/", h.Create)
	routes.Get("/", h.List)
	routes.Get("/:id", h.GetByID)
	routes.Put("/:id", h.Update)
	routes.Delete("/:id", h.Delete)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req Create${MODULE_TITLE}Request
	if err := validator.ParseAndValidate(c, &req); err != nil {
		return response.BadRequest(c, err.Error())
	}
	
	res, err := h.service.Create(c.Context(), &req)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.Created(c, res)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	res, total, err := h.service.List(c.Context(), limit, offset)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.OKWithMeta(c, res, map[string]interface{}{"total": total, "limit": limit, "offset": offset})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Invalid ID")
	}

	res, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.OK(c, res)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Invalid ID")
	}

	var req Update${MODULE_TITLE}Request
	if err := validator.ParseAndValidate(c, &req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	res, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.OK(c, res)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "Invalid ID")
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.OK(c, map[string]string{"message": "deleted"})
}
EOF

# 5. Create module.go
cat > "$MODULE_DIR/module.go" <<EOF
package $MODULE_LOWER

import (
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type Module struct {
	handler *Handler
}

func New(db bun.IDB) *Module {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)
	return &Module{handler: handler}
}

func (m *Module) Register(r fiber.Router) {
	m.handler.RegisterRoutes(r)
}
EOF

echo "Success: Full CRUD Module '$MODULE_LOWER' generated successfully at $MODULE_DIR/"
