package auth

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"golang-base/internal/domain"
	"golang-base/internal/middleware"
	"golang-base/internal/pkg/response"
	vld "golang-base/internal/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type handler struct {
	svc *service
}

func NewHandler(svc *service) *handler {
	return &handler{
		svc: svc,
	}
}

func (h *handler) RegisterRoutes(router fiber.Router) {
	// Auth routes (public)
	auth := router.Group("/auth")
	// auth.Post("/register", h.Register) // disabled: user creation via POST /users (super_admin only)
	auth.Post("/login", h.Login)
	auth.Post("/refresh", h.RefreshToken)

	// Protected auth routes
	authProtected := auth.Group("", middleware.AuthMiddleware())
	authProtected.Get("/me", h.Me)

	// User routes
	users := router.Group("/users", middleware.AuthMiddleware())
	users.Post("/", middleware.AllowedPermissions("user.create"), h.CreateUser)
	users.Get("/", middleware.AllowedPermissions("user.view"), h.ListUsers)
	users.Get("/:id", middleware.AllowedPermissions("user.view"), h.GetUser)
	users.Patch("/:id", middleware.AllowedPermissions("user.edit"), h.UpdateUser)
	users.Delete("/:id", middleware.AllowedPermissions("user.delete"), h.DeleteUser)

	// User permissions
	users.Get("/:user_id/permissions", middleware.AllowedPermissions("user.view"), h.GetUserPermissions)
	users.Put("/:user_id/permissions/:permission", middleware.AllowedPermissions("user.edit"), h.GrantUserPermission)
	users.Delete("/:user_id/permissions/:permission", middleware.AllowedPermissions("user.edit"), h.RevokeUserPermission)

	// Roles
	roles := router.Group("/roles", middleware.AuthMiddleware())
	roles.Get("/", middleware.AllowedPermissions("user.view"), h.GetAllRoles)
	roles.Get("/:role/permissions", middleware.AllowedPermissions("user.view"), h.GetRolePermissions)
	roles.Put("/:role/permissions", middleware.AllowedPermissions("user.edit"), h.UpdateRolePermissions)

	// Permissions (protected)
	permissions := router.Group("/permissions", middleware.AuthMiddleware())
	permissions.Get("/matrix", middleware.AllowedPermissions("user.view"), h.GetPermissionMatrix)
	permissions.Get("/list", h.ListPermissions)

	// Audit logs
	audit := router.Group("/audit-logs", middleware.AuthMiddleware())
	audit.Get("/permission-changes", middleware.AllowedPermissions("user.view"), h.QueryPermissionChanges)
}

// Auth Handlers

func (h *handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if errs := vld.ValidateStruct(&req); errs != nil {
		return response.BadRequestValidation(c, errs)
	}

	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{"user"} // default role
	}

	user, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name, roles)
	if err != nil {
		if err == ErrUserExists {
			return response.BadRequest(c, "user already exists")
		}
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.Created(c, ToUserResponse(user))
}

func (h *handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if errs := vld.ValidateStruct(&req); errs != nil {
		return response.BadRequestValidation(c, errs)
	}

	accessToken, refreshToken, user, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return response.Unauthorized(c, "invalid credentials")
	}

	return response.OK(c, NewAuthResponse(accessToken, refreshToken, ToUserResponsePtr(user)))
}

func (h *handler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	newAccessToken, err := h.svc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return response.Unauthorized(c, "invalid refresh token")
	}

	return response.OK(c, fiber.Map{
		"access_token": newAccessToken,
		"token_type":   "Bearer",
		"expires_in":   900, // 15 minutes in seconds
	})
}

func (h *handler) Me(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	user, err := h.svc.GetUser(c.Context(), userID)
	if err != nil {
		return response.NotFound(c, "user not found")
	}

	// Get role permissions and user permission overrides
	permissions, err := h.svc.GetComputedPermissions(c.Context(), userID, user.Roles)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	userResp := ToUserResponseWithPermissions(user, permissions)
	return response.OK(c, userResp)
}

// User Handlers

func (h *handler) ListUsers(c *fiber.Ctx) error {
	var filter UserFilter
	if err := c.QueryParser(&filter); err != nil {
		return response.BadRequest(c, "invalid query parameters")
	}

	page, limit, offset := response.ParsePaginationParams(c)
	filter.Limit = limit
	filter.Offset = offset

	users, total, err := h.svc.ListUsers(c.Context(), &filter)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	userResponses := make([]UserResponse, len(users))
	for i, u := range users {
		permissions, err := h.svc.GetComputedPermissions(c.Context(), u.ID, u.Roles)
		if err != nil {
			permissions = []string{}
		}
		userResponses[i] = ToUserResponseWithPermissions(u, permissions)
	}

	return response.PaginatedResponse(c, userResponses, page, limit, total)
}

func (h *handler) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")

	user, err := h.svc.GetUser(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "user not found")
	}

	permissions, err := h.svc.GetComputedPermissions(c.Context(), id, user.Roles)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, ToUserResponseWithPermissions(user, permissions))
}

func (h *handler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	var name *string
	if req.Name != "" {
		name = &req.Name
	}
	var roles []string
	if len(req.Roles) > 0 {
		roles = req.Roles
	}
	var isActive *bool
	if req.IsActive != nil {
		isActive = req.IsActive
	}

	user, err := h.svc.UpdateUser(c.Context(), id, name, roles, isActive)
	if err != nil {
		if err == ErrUserNotFound {
			return response.NotFound(c, "user not found")
		}
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	permissions, err := h.svc.GetComputedPermissions(c.Context(), user.ID, user.Roles)
	if err != nil {
		permissions = []string{}
	}

	return response.OK(c, ToUserResponseWithPermissions(user, permissions))
}

func (h *handler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	// Check if user exists
	if _, err := h.svc.GetUser(c.Context(), id); err != nil {
		return response.NotFound(c, "user not found")
	}

	if err := h.svc.DeleteUser(c.Context(), id); err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, fiber.Map{"message": "User deactivated successfully"})
}

// CreateUser creates a new user (requires user.create permission)
func (h *handler) CreateUser(c *fiber.Ctx) error {
	actorID := c.Locals("userID").(string)
	if !h.svc.HasPermission(c.Context(), actorID, "user.create") {
		return response.Forbidden(c, "missing permission: user.create")
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if errs := vld.ValidateStruct(&req); errs != nil {
		return response.BadRequestValidation(c, errs)
	}

	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	roleNames, _ := h.svc.GetAllRoleNames(c.Context())
	if len(roleNames) > 0 {
		for _, r := range roles {
			if !slices.Contains(roleNames, r) {
				return response.BadRequest(c, fmt.Sprintf("invalid role: must be one of %s", strings.Join(roleNames, ", ")))
			}
		}
	}

	user, err := h.svc.CreateUser(c.Context(), req.Email, req.Password, req.Name, roles)
	if err != nil {
		if err == ErrUserExists {
			return response.BadRequest(c, "user already exists")
		}
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	permissions, err := h.svc.GetComputedPermissions(c.Context(), user.ID, user.Roles)
	if err != nil {
		permissions = []string{}
	}

	return response.Created(c, ToUserResponseWithPermissions(user, permissions))
}

// Role Handlers

func (h *handler) GetRolePermissions(c *fiber.Ctx) error {
	role := c.Params("role")

	// Validate role exists
	roleNames, _ := h.svc.GetAllRoleNames(c.Context())
	if len(roleNames) > 0 && !slices.Contains(roleNames, role) {
		return response.NotFound(c, "role not found")
	}

	perms, err := h.svc.GetRolePermissions(c.Context(), role)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, fiber.Map{"permissions": perms})
}

func (h *handler) UpdateRolePermissions(c *fiber.Ctx) error {
	role := c.Params("role")

	var req UpdatePermissionsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if err := h.svc.UpdateRolePermissions(c.Context(), role, req.Permissions); err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, fiber.Map{"message": "permissions updated"})
}

// Permission Handlers

// GetPermissionMatrix returns hardcoded roles & permissions with DB-based role permission assignments
func (h *handler) GetPermissionMatrix(c *fiber.Ctx) error {
	matrix, err := h.svc.GetPermissionMatrix(c.Context())
	if err != nil {
		return response.InternalError(c, "failed to get permission matrix")
	}

	return response.OK(c, matrix)
}

// ListPermissions returns all available permission definitions and roles (static from code)
func (h *handler) ListPermissions(c *fiber.Ctx) error {
	result, err := h.svc.ListPermissions(c.Context())
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, result)
}

// GetAllRoles returns all roles with their permissions from DB
func (h *handler) GetAllRoles(c *fiber.Ctx) error {
	roleNames, _ := h.svc.GetAllRoleNames(c.Context())

	dbPerms, err := h.svc.GetAllRolesWithPermissions(c.Context())
	if err != nil {
		dbPerms = make(map[string][]string)
	}

	result := make([]RolePermissionsResponse, 0, len(roleNames))
	for _, role := range roleNames {
		perms := dbPerms[role]
		if perms == nil {
			perms = []string{}
		}
		result = append(result, RolePermissionsResponse{
			Role:        role,
			Permissions: perms,
		})
	}

	return response.OK(c, result)
}

func (h *handler) GetUserPermissions(c *fiber.Ctx) error {
	userID := c.Params("user_id")

	perms, err := h.svc.GetUserPermissions(c.Context(), userID)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, fiber.Map{"permissions": perms})
}

func parseExpiresAt(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, *s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid expires_at format: %s", *s)
}

func (h *handler) GrantUserPermission(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	permission := c.Params("permission")
	if permission == "" {
		return response.BadRequest(c, "permission is required")
	}

	actorID := c.Locals("userID").(string)

	var req UserPermissionRequest
	_ = c.BodyParser(&req)

	isGranted := true
	if req.Granted != nil {
		isGranted = *req.Granted
	} else if g := c.Query("granted"); g == "false" || g == "0" {
		isGranted = false
	}

	var reasonPtr *string
	if req.Reason != nil && *req.Reason != "" {
		reasonPtr = req.Reason
	} else if r := c.Query("reason"); r != "" {
		reasonPtr = &r
	}

	var expiresAtPtr *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := parseExpiresAt(req.ExpiresAt)
		if err != nil {
			return response.BadRequest(c, err.Error())
		}
		expiresAtPtr = t
	} else if exp := c.Query("expires_at"); exp != "" {
		t, err := parseExpiresAt(&exp)
		if err != nil {
			return response.BadRequest(c, err.Error())
		}
		expiresAtPtr = t
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	if err := h.svc.GrantUserPermission(c.Context(), "user_permission", nil, &userID, permission, isGranted, expiresAtPtr, actorID, reasonPtr, &ipAddress, &userAgent); err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	msg := "permission granted"
	if !isGranted {
		msg = "permission revoked"
	}
	return response.OK(c, fiber.Map{"message": msg})
}

func (h *handler) RevokeUserPermission(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	permission := c.Params("permission")
	if permission == "" {
		return response.BadRequest(c, "permission is required")
	}

	actorID := c.Locals("userID").(string)
	reason := c.Query("reason")
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	if err := h.svc.RevokeUserPermission(c.Context(), "user_permission", nil, &userID, permission, actorID, reasonPtr, &ipAddress, &userAgent); err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	return response.OK(c, fiber.Map{"message": "permission revoked"})
}

// Audit Handlers

func (h *handler) QueryPermissionChanges(c *fiber.Ctx) error {
	var filter domain.PermissionQueryFilter

	if targetType := c.Query("target_type"); targetType != "" {
		filter.TargetType = &targetType
	}
	if targetRole := c.Query("target_role"); targetRole != "" {
		filter.TargetRole = &targetRole
	}
	if targetUserID := c.Query("target_user_id"); targetUserID != "" {
		filter.TargetUserID = &targetUserID
	}
	if permission := c.Query("permission"); permission != "" {
		filter.Permission = &permission
	}
	if changedBy := c.Query("changed_by"); changedBy != "" {
		filter.ChangedBy = &changedBy
	}
	if action := c.Query("action"); action != "" {
		filter.Action = &action
	}
	if fromDate := c.Query("from_date"); fromDate != "" {
		if t, err := time.Parse("2006-01-02", fromDate); err == nil {
			filter.FromDate = &t
		}
	}
	if toDate := c.Query("to_date"); toDate != "" {
		if t, err := time.Parse("2006-01-02", toDate); err == nil {
			filter.ToDate = &t
		}
	}
	page, limit, offset := response.ParsePaginationParams(c)
	filter.Limit = limit
	filter.Offset = offset

	logs, err := h.svc.QueryPermissionChanges(c.Context(), &filter)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	total, err := h.svc.CountPermissionChanges(c.Context(), &filter)
	if err != nil {
		total = 0
	}

	return response.PaginatedResponse(c, logs, page, limit, total)
}
