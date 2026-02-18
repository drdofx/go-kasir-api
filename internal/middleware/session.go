package middleware

import (
	"context"
	"net/http"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/service"
)

type userContextKey struct{}

// SessionAuth validates the session_id cookie and stores the user in context.
func SessionAuth(authService *service.AuthService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil || cookie.Value == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := authService.ValidateSession(cookie.Value)
			if err != nil {
				http.Error(w, "Session expired", http.StatusUnauthorized)
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
