package pkg

import (
	"os"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestNewJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "")
	jwt := NewJWT()

	assert.NotNil(t, jwt)
	assert.Equal(t, 15, jwt.AccessExpireMin)
	assert.Equal(t, 7, jwt.RefreshExpireDay)
	assert.NotEmpty(t, jwt.Secret)
}

func TestNewJWT_WithSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key")
	defer os.Unsetenv("JWT_SECRET")

	jwt := NewJWT()
	assert.Equal(t, "test-secret-key", jwt.Secret)
}

func TestGenerateAccessToken(t *testing.T) {
	jwt := &JWT{
		Secret:          "test-secret",
		AccessExpireMin: 15,
	}

	token, err := jwt.GenerateAccessToken("user-123")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token structure
	parsed, err := jwt.Parse(token, "test-secret")
	assert.NoError(t, err)
	assert.True(t, parsed.Valid)
}

func TestGenerateRefreshToken(t *testing.T) {
	jwt := &JWT{
		Secret:           "test-secret",
		RefreshExpireDay: 7,
	}

	token, err := jwt.GenerateRefreshToken("user-123")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token structure
	parsed, err := jwt.Parse(token, "test-secret")
	assert.NoError(t, err)
	assert.True(t, parsed.Valid)
}

func TestGenerateToken_DifferentExpiry(t *testing.T) {
	shortJWT := &JWT{Secret: "test", AccessExpireMin: 1, RefreshExpireDay: 1}
	longJWT := &JWT{Secret: "test", AccessExpireMin: 60, RefreshExpireDay: 30}

	shortToken, _ := shortJWT.GenerateAccessToken("user-1")
	longToken, _ := longJWT.GenerateAccessToken("user-1")

	// Both should be valid tokens
	assert.NotEmpty(t, shortToken)
	assert.NotEmpty(t, longToken)
}

func TestValidateToken_Success(t *testing.T) {
	// Create JWT with long expiry for test
	jwt := &JWT{Secret: "test-secret", AccessExpireMin: 60}

	token, err := jwt.GenerateAccessToken("user-123")
	assert.NoError(t, err)

	claims, err := jwt.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	jwt := &JWT{Secret: "test-secret"}

	_, err := jwt.ValidateToken("invalid-token")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	jwt1 := &JWT{Secret: "secret-1"}
	jwt2 := &JWT{Secret: "secret-2"}

	token, _ := jwt1.GenerateAccessToken("user-123")

	_, err := jwt2.ValidateToken(token)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestValidateToken_RefreshToken(t *testing.T) {
	jwt := &JWT{Secret: "test-secret", RefreshExpireDay: 7}

	refreshToken, _ := jwt.GenerateRefreshToken("user-123")

	claims, err := jwt.ValidateToken(refreshToken)

	assert.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "refresh", claims.Subject)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create JWT with 0 minute expiry (expires immediately)
	jwt := &JWT{Secret: "test-secret", AccessExpireMin: 0}

	token, _ := jwt.GenerateAccessToken("user-123")

	_, err := jwt.ValidateToken(token)

	// Token should be expired or invalid
	assert.Error(t, err)
}

// Helper for parsing tokens in tests
func (j *JWT) Parse(tokenString string, secret string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
}
