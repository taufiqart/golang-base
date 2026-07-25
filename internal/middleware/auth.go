package middleware

import (
	"strings"
	"sync"

	jwtpkg "golang-base/internal/pkg/jwt"
	"golang-base/internal/pkg/response"

	"github.com/gofiber/fiber/v2"
)

var (
	jwtInstance *jwtpkg.JWT
	jwtOnce     sync.Once
)

func getJWT() *jwtpkg.JWT {
	jwtOnce.Do(func() {
		jwtInstance = jwtpkg.NewJWT()
	})
	return jwtInstance
}

// AuthMiddleware validates JWT token and sets userID in context
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Unauthorized(c, "missing authorization header")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return response.Unauthorized(c, "invalid authorization header format")
		}

		tokenString := authHeader[7:]
		claims, err := getJWT().ValidateToken(tokenString)
		if err != nil {
			return response.Unauthorized(c, "invalid or expired token")
		}

		c.Locals("userID", claims.UserID)
		return c.Next()
	}
}
