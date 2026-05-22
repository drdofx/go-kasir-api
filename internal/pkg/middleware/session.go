package middleware

import (
	"context"
	"net/http"
	"strings"
)

type ContextUser struct {
	ID        int
	Username  string
	Name      string
	Role      string
	CreatedAt string
}

type contextKey string

const userContextKey contextKey = "user"

type tokenValidator interface {
	ValidateToken(tokenStr string) (int, string, string, error)
}

func JWTAuth(validator tokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"message":"unauthorized","code":401}`, http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"message":"unauthorized","code":401}`, http.StatusUnauthorized)
				return
			}
			userID, username, role, err := validator.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"message":"unauthorized","code":401}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, &ContextUser{
				ID:       userID,
				Username: username,
				Role:     role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *ContextUser {
	user, _ := ctx.Value(userContextKey).(*ContextUser)
	return user
}
