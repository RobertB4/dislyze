package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
)

// Claims represents the JWT claims
type Claims struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates the JWT token and adds tenant_id and user_id to the context
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the access token from the cookie
			cookie, err := r.Cookie("access_token")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse and validate the token
			token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (any, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get the claims
			claims, ok := token.Claims.(*Claims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Add tenant_id and user_id to the context
			ctx := context.WithValue(r.Context(), TenantIDKey, claims.TenantID)
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
