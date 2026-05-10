package middleware

import (
	"context"
	"net/http"
	"strings"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/service"
)

type userContextKey struct{}

// JWTAuth validates the Authorization Bearer token and stores the user in context.
func JWTAuth(authService *service.AuthService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			user, err := authService.GetUserFromToken(r.Context(), claims)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext returns the authenticated user from context.
func UserFromContext(ctx context.Context) *model.User {
	u, _ := ctx.Value(userContextKey{}).(*model.User)
	return u
}
