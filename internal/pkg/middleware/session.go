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
	OrgID     int
	BranchID  *int
	CreatedAt string
}

type contextKey string

const userContextKey contextKey = "user"
const orgContextKey contextKey = "org_id"

type tokenValidator interface {
	ValidateToken(tokenStr string) (int, string, string, int, *int, error)
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
			userID, username, role, orgID, branchID, err := validator.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"message":"unauthorized","code":401}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, &ContextUser{
				ID:       userID,
				Username: username,
				Role:     role,
				OrgID:    orgID,
				BranchID: branchID,
			})
			ctx = context.WithValue(ctx, orgContextKey, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *ContextUser {
	user, _ := ctx.Value(userContextKey).(*ContextUser)
	return user
}

func OrgIDFromContext(ctx context.Context) int {
	if orgID, ok := ctx.Value(orgContextKey).(int); ok {
		return orgID
	}
	return 0
}
