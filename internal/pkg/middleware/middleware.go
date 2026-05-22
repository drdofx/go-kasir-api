package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Middleware func(http.Handler) http.Handler

const requestIDHeader = "X-Request-Id"

func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func CORS(allowedOrigin string) Middleware {
	origin := strings.TrimSpace(allowedOrigin)
	if origin == "" {
		origin = "http://localhost:8080"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func Logger(level string) Middleware {
	lvl, err := zerolog.ParseLevel(strings.ToLower(strings.TrimSpace(level)))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	httpLogger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
	).With().Timestamp().Logger()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			httpLogger.Info().
				Str("request_id", requestIDFromContext(r)).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rw.statusCode).
				Dur("duration", time.Since(start)).
				Str("client_ip", clientIP(r)).
				Str("user_agent", r.UserAgent()).
				Msg("http_request")
		})
	}
}

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := strings.TrimSpace(r.Header.Get(requestIDHeader))
			if reqID == "" {
				reqID = newRequestID()
			}
			w.Header().Set(requestIDHeader, reqID)
			next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), reqID)))
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}

	hostPort := strings.TrimSpace(r.RemoteAddr)
	if hostPort == "" {
		return ""
	}

	lastColon := strings.LastIndex(hostPort, ":")
	if lastColon == -1 {
		return hostPort
	}
	if strings.Count(hostPort, ":") > 1 && strings.HasPrefix(hostPort, "[") {
		return strings.Trim(hostPort[:lastColon], "[]")
	}
	return hostPort[:lastColon]
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
