package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type AuthHandler struct {
	service *AuthService
}

func NewAuthHandler(service *AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Message      string   `json:"message"`
	Token        string   `json:"token"` // Backward-compatible alias for access_token.
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         userJSON `json:"user"`
}

type userJSON struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	OrgID       int      `json:"org_id"`
	BranchID    *int     `json:"branch_id,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type messageResponse struct {
	Message string `json:"message"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helpers.RequireNonEmpty(map[string]string{"username": req.Username, "password": req.Password}); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	tokens, user, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			helpers.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, loginResponse{
		Message:      "login successful",
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: userJSON{
			ID:          user.ID,
			Username:    user.Username,
			Name:        user.Name,
			Role:        user.Role,
			Permissions: user.Permissions,
			OrgID:       user.OrganizationID,
			BranchID:    user.BranchID,
			CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if r.Body != nil {
		_ = helpers.DecodeJSON(r, &req)
	}
	if err := h.service.Logout(req.RefreshToken); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, messageResponse{Message: "logout successful"})
}

func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := helpers.RequireNonEmpty(map[string]string{"refresh_token": req.RefreshToken}); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	accessToken, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			helpers.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"access_token": accessToken, "token": accessToken})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, userJSON{
		ID:          user.ID,
		Username:    user.Username,
		Name:        user.Name,
		Role:        user.Role,
		Permissions: user.Permissions,
		OrgID:       user.OrgID,
		BranchID:    user.BranchID,
		CreatedAt:   user.CreatedAt,
	})
}

func (h *AuthHandler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := h.service.FindAllUsers()
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		pagination, err := helpers.ParsePagination(r)
		if err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		helpers.WritePaginationHeaders(w, pagination, len(users))
		users = helpers.Paginate(users, pagination)
		res := make([]userJSON, len(users))
		for i, u := range users {
			res[i] = userJSON{
				ID: u.ID, Username: u.Username, Name: u.Name,
				Role: u.Role, Permissions: u.Permissions, OrgID: u.OrganizationID, BranchID: u.BranchID,
				CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}
		helpers.WriteJSON(w, http.StatusOK, res)
	case http.MethodPost:
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Name     string `json:"name"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			helpers.WriteError(w, http.StatusBadRequest, "username and password are required")
			return
		}
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		u, err := h.service.CreateUser(req.Username, req.Password, req.Name, req.Role, user.OrgID, user.BranchID)
		if err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		helpers.WriteJSON(w, http.StatusCreated, userJSON{
			ID: u.ID, Username: u.Username, Name: u.Name,
			Role: u.Role, Permissions: u.Permissions, OrgID: u.OrganizationID, BranchID: u.BranchID,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *AuthHandler) HandleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.UpdateUserRole(id, req.Role, req.Permissions); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"message": "role updated"})
}

func (h *AuthHandler) HandleSwitchBranch(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		BranchID int `json:"branch_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BranchID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "branch_id is required")
		return
	}
	token, err := h.service.SwitchBranch(user.ID, req.BranchID)
	if err != nil {
		if errors.Is(err, ErrBranchNotFound) {
			helpers.WriteError(w, http.StatusNotFound, "branch not found")
			return
		}
		if errors.Is(err, ErrBranchNotAllowed) {
			helpers.WriteError(w, http.StatusForbidden, "branch does not belong to user's organization")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		helpers.WriteError(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}
	if len(req.NewPassword) < 6 {
		helpers.WriteError(w, http.StatusBadRequest, "new password must be at least 6 characters")
		return
	}
	if err := h.service.ChangePassword(user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrIncorrectPassword) {
			helpers.WriteError(w, http.StatusBadRequest, "current password is incorrect")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, messageResponse{Message: "password changed successfully"})
}
