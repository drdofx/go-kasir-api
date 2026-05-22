package category

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
)

type CategoryHandler struct {
	service *CategoryService
}

func NewCategoryHandler(service *CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

type categoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type categoryResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toResponse(c Category) categoryResponse {
	return categoryResponse{ID: c.ID, Name: c.Name, Description: c.Description}
}

func toResponses(categories []Category) []categoryResponse {
	res := make([]categoryResponse, len(categories))
	for i, c := range categories {
		res[i] = toResponse(c)
	}
	return res
}

func (h *CategoryHandler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *CategoryHandler) HandleCategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		helpers.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getByID(w, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, id)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.FindAll()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if categories == nil {
		categories = []Category{}
	}
	helpers.WriteJSON(w, http.StatusOK, toResponses(categories))
}

func (h *CategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c := &Category{Name: req.Name, Description: req.Description}
	if err := h.service.Create(c); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, toResponse(*c))
}

func (h *CategoryHandler) getByID(w http.ResponseWriter, id int) {
	c, err := h.service.FindByID(id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if c == nil {
		helpers.WriteError(w, http.StatusNotFound, "category not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toResponse(*c))
}

func (h *CategoryHandler) update(w http.ResponseWriter, r *http.Request, id int) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c := &Category{ID: id, Name: req.Name, Description: req.Description}
	if err := h.service.Update(c); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toResponse(*c))
}

func (h *CategoryHandler) delete(w http.ResponseWriter, id int) {
	if err := h.service.Delete(id); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
