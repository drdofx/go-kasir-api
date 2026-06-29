package product

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"
)

type ProductHandler struct {
	service *ProductService
}

func NewProductHandler(service *ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

type productRequest struct {
	Name       string `json:"name"`
	Price      int    `json:"price"`
	CategoryID *int   `json:"category_id"`
}

type productStockResponse struct {
	BranchID   int    `json:"branch_id"`
	BranchName string `json:"branch_name,omitempty"`
	Stock      int    `json:"stock"`
}

type productResponse struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	Price        int                    `json:"price"`
	CategoryID   *int                   `json:"category_id"`
	CategoryName string                 `json:"category_name"`
	Stocks       []productStockResponse `json:"stocks,omitempty"`
}

func toResponse(p Product) productResponse {
	return productResponse{
		ID:           p.ID,
		Name:         p.Name,
		Price:        p.Price,
		CategoryID:   p.CategoryID,
		CategoryName: p.CategoryName,
	}
}

func toResponses(products []Product) []productResponse {
	res := make([]productResponse, len(products))
	for i, p := range products {
		res[i] = toResponse(p)
	}
	return res
}

func (h *ProductHandler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ProductHandler) HandleProductByID(w http.ResponseWriter, r *http.Request) {
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
		h.getByID(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	name := r.URL.Query().Get("name")
	products, err := h.service.FindAllForOrg(user.OrgID, name)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if products == nil {
		products = []Product{}
	}
	pagination, err := helpers.ParsePagination(r)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WritePaginationHeaders(w, pagination, len(products))
	products = helpers.Paginate(products, pagination)
	helpers.WriteJSON(w, http.StatusOK, toResponses(products))
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p := &Product{Name: req.Name, Price: req.Price, CategoryID: req.CategoryID}
	if err := h.service.CreateForOrg(user.OrgID, p); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, toResponse(*p))
}

func (h *ProductHandler) getByID(w http.ResponseWriter, r *http.Request, id int) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	p, err := h.service.FindByIDForOrg(user.OrgID, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if p == nil {
		helpers.WriteError(w, http.StatusNotFound, "product not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toResponse(*p))
}

func (h *ProductHandler) update(w http.ResponseWriter, r *http.Request, id int) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p := &Product{ID: id, Name: req.Name, Price: req.Price, CategoryID: req.CategoryID}
	if err := h.service.UpdateForOrg(user.OrgID, p); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, toResponse(*p))
}

func (h *ProductHandler) delete(w http.ResponseWriter, r *http.Request, id int) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.DeleteForOrg(user.OrgID, id); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
