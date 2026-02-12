package middleware

import "context"

type requestIDContextKey struct{}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(r interface{ Context() context.Context }) string {
	if r == nil {
		return ""
	}
	if id, ok := r.Context().Value(requestIDContextKey{}).(string); ok {
		return id
	}
	return ""
}
