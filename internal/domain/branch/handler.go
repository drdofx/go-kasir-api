package branch

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type BranchHandler struct {
	service *BranchService
}

func NewBranchHandler(service *BranchService) *BranchHandler {
	return &BranchHandler{service: service}
}

type branchRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
}

type branchResponse struct {
	ID             int    `json:"id"`
	OrganizationID int    `json:"organization_id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	Address        string `json:"address"`
	CreatedAt      string `json:"created_at"`
}

func toResponse(b Branch) branchResponse {
	return branchResponse{
		ID: b.ID, OrganizationID: b.OrganizationID,
		Name: b.Name, Code: b.Code, Address: b.Address,
		CreatedAt: b.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toResponses(bs []Branch) []branchResponse {
	res := make([]branchResponse, len(bs))
	for i, b := range bs {
		res[i] = toResponse(b)
	}
	return res
}

func (h *BranchHandler) HandleBranches(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	switch r.Method {
	case http.MethodGet:
		bs, err := h.service.FindByOrgID(user.OrgID)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if bs == nil {
			bs = []Branch{}
		}
		helpers.WriteJSON(w, http.StatusOK, toResponses(bs))
	case http.MethodPost:
		var req branchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		b := &Branch{OrganizationID: user.OrgID, Name: req.Name, Code: req.Code, Address: req.Address}
		if err := h.service.Create(b); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		helpers.WriteJSON(w, http.StatusCreated, toResponse(*b))
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *BranchHandler) HandleBranchByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req branchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		b := &Branch{ID: id, Name: req.Name, Code: req.Code, Address: req.Address}
		if err := h.service.Update(b); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		b, err = h.service.repo.FindByID(id)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if b == nil {
			helpers.WriteError(w, http.StatusNotFound, "branch not found")
			return
		}
		helpers.WriteJSON(w, http.StatusOK, toResponse(*b))
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
