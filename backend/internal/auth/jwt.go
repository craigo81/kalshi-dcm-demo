// Package auth provides JWT-based authentication for the DCM demo.
// Core Principle 17: Access controls for fitness standards.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// =============================================================================
// CONFIGURATION
// =============================================================================

var (
	// In production, use env var or secrets manager
	jwtSecret = []byte("dcm-demo-secret-key-change-in-production")
	jwtIssuer = "kalshi-dcm-demo"

	ErrInvalidToken = errors.New("invalid or expired token")
	ErrMissingToken = errors.New("missing authorization token")
)

// Claims represents JWT claims for user sessions.
type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	Verified  bool   `json:"verified"`
	jwt.RegisteredClaims
}

// ContextKey for storing user info in request context.
type ContextKey string

const (
	UserContextKey ContextKey = "user"
)

// =============================================================================
// TOKEN GENERATION
// =============================================================================

// GenerateToken creates a new JWT for authenticated users.
// Core Principle 17: Authenticates participants.
func GenerateToken(userID, email, status string, verified bool) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Email:    email,
		Status:   status,
		Verified: verified,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken verifies and parses a JWT.
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

// AuthMiddleware validates JWT and adds user context.
// Core Principle 17: Enforces access controls.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"success":false,"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"success":false,"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			http.Error(w, `{"success":false,"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireVerified ensures user has completed KYC.
// Core Principle 17: Fitness standards for trading.
func RequireVerified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil {
			http.Error(w, `{"success":false,"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if !claims.Verified {
			http.Error(w, `{"success":false,"error":"KYC verification required","code":"KYC_REQUIRED"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts user claims from request context.
func GetUserFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// GetClientIP extracts client IP for audit logging.
// Core Principle 18: IP tracking for audit trail.
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (when behind proxy)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP (original client)
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
