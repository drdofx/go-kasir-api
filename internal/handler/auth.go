package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-kasir-api/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Message string      `json:"message"`
	Token   string      `json:"token"`
	User    interface{} `json:"user"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	token, user, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	jsonResponse(w, http.StatusOK, loginResponse{
		Message: "Login successful",
		Token:   token,
		User:    user,
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// JWT logout is client-side: client discards the token.
	// No server action needed for stateless JWT.
	jsonResponse(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Extract Bearer token
	parts := make([]string, 0)
	for _, p := range strings.SplitN(authHeader, " ", 2) {
		parts = append(parts, p)
	}
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}

	claims, err := h.authService.ValidateToken(parts[1])
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	user, err := h.authService.GetUserFromToken(r.Context(), claims)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	jsonResponse(w, http.StatusOK, user)
}
