package supplier

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
)

type SupplierHandler struct {
	service *SupplierService
}

func NewSupplierHandler(service *SupplierService) *SupplierHandler {
	return &SupplierHandler{service: service}
}

type supplierRequest struct {
	Name          string `json:"name"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Address       string `json:"address"`
}

type supplierResponse struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Address       string `json:"address"`
	CreatedAt     string `json:"created_at"`
}

func toResponse(s Supplier) supplierResponse {
	return supplierResponse{
		ID: s.ID, Name: s.Name, ContactPerson: s.ContactPerson,
		Phone: s.Phone, Email: s.Email, Address: s.Address,
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toResponses(ss []Supplier) []supplierResponse {
	res := make([]supplierResponse, len(ss))
	for i, s := range ss {
		res[i] = toResponse(s)
	}
	return res
}

func (h *SupplierHandler) HandleSuppliers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		search := r.URL.Query().Get("search")
		ss, err := h.service.FindAll(search)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if ss == nil {
			ss = []Supplier{}
		}
		helpers.WriteJSON(w, http.StatusOK, toResponses(ss))
	case http.MethodPost:
		var req supplierRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		s := &Supplier{
			Name: req.Name, ContactPerson: req.ContactPerson,
			Phone: req.Phone, Email: req.Email, Address: req.Address,
		}
		if err := h.service.Create(s); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		helpers.WriteJSON(w, http.StatusCreated, toResponse(*s))
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *SupplierHandler) HandleSupplierByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s, err := h.service.FindByID(id)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if s == nil {
			helpers.WriteError(w, http.StatusNotFound, "supplier not found")
			return
		}
		helpers.WriteJSON(w, http.StatusOK, toResponse(*s))
	case http.MethodPut:
		var req supplierRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		s := &Supplier{
			ID: id, Name: req.Name, ContactPerson: req.ContactPerson,
			Phone: req.Phone, Email: req.Email, Address: req.Address,
		}
		if err := h.service.Update(s); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		helpers.WriteJSON(w, http.StatusOK, toResponse(*s))
	case http.MethodDelete:
		if err := h.service.Delete(id); err != nil {
			helpers.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
