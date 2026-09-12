package socketio

import (
	"errors"
	"strings"

	jwtpkg "golang-base/internal/pkg/jwt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zishang520/socket.io/v2/socket"
)

// ExtractToken attempts to extract a JWT token string from handshake auth, query, or headers.
func ExtractToken(handshake *socket.Handshake) string {
	if handshake == nil {
		return ""
	}

	// 1. Try Handshake.Auth (e.g. io(..., { auth: { token: "..." } }))
	if authMap, ok := handshake.Auth.(map[string]any); ok {
		if tokenVal, exists := authMap["token"]; exists {
			if tokenStr, ok := tokenVal.(string); ok && tokenStr != "" {
				return cleanBearer(tokenStr)
			}
		}
		if tokenVal, exists := authMap["accessToken"]; exists {
			if tokenStr, ok := tokenVal.(string); ok && tokenStr != "" {
				return cleanBearer(tokenStr)
			}
		}
	}

	// 2. Try Handshake.Headers (e.g. extraHeaders: { Authorization: "Bearer ..." })
	if authHeaders, ok := handshake.Headers["authorization"]; ok && len(authHeaders) > 0 {
		return cleanBearer(authHeaders[0])
	}
	if authHeaders, ok := handshake.Headers["Authorization"]; ok && len(authHeaders) > 0 {
		return cleanBearer(authHeaders[0])
	}

	// 3. Try Handshake.Query (e.g. io(..., { query: { token: "..." } }))
	if tokenQuery, ok := handshake.Query["token"]; ok && len(tokenQuery) > 0 {
		return cleanBearer(tokenQuery[0])
	}

	return ""
}

func cleanBearer(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}

// AuthenticateHandshake extracts and validates the JWT token from a handshake.
func AuthenticateHandshake(handshake *socket.Handshake, secret string) (*jwtpkg.JWTClaims, error) {
	tokenStr := ExtractToken(handshake)
	if tokenStr == "" {
		return nil, errors.New("missing token")
	}

	claims := &jwtpkg.JWTClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil || !parsedToken.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}

// JWTMiddleware creates a Socket.IO namespace middleware that authenticates connections via JWT.
func JWTMiddleware(secret string) socket.NamespaceMiddleware {
	return func(client *socket.Socket, next func(*socket.ExtendedError)) {
		claims, err := AuthenticateHandshake(client.Handshake(), secret)
		if err != nil {
			next(socket.NewExtendedError("Authentication error: "+err.Error(), nil))
			return
		}

		// Attach verified claims to socket data for access in event listeners
		client.SetData(claims)
		next(nil)
	}
}

// GetClaims retrieves verified JWTClaims attached to a connected socket.
func GetClaims(client *socket.Socket) (*jwtpkg.JWTClaims, bool) {
	if client == nil || client.Data() == nil {
		return nil, false
	}
	claims, ok := client.Data().(*jwtpkg.JWTClaims)
	return claims, ok
}

// GetUserID retrieves the user ID from the verified JWT claims on the socket.
func GetUserID(client *socket.Socket) (string, bool) {
	claims, ok := GetClaims(client)
	if !ok || claims == nil {
		return "", false
	}
	return claims.UserID, true
}
