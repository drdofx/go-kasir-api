package product

import (
	"encoding/json"
	"net/http"
    "strconv"

    "go-kasir-api/internal/pkg/helpers"
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
    Stock      int    `json:"stock"`
    CategoryID *int   `json:"category_id"`
}

type productResponse struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    Price        int    `json:"price"`
    Stock        int    `json:"stock"`
    CategoryID   *int   `json:"category_id"`
    CategoryName string `json:"category_name"`
}

func toResponse(p Product) productResponse {
    return productResponse{
        ID:           p.ID,
        Name:         p.Name,
        Price:        p.Price,
        Stock:        p.Stock,
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
        h.getByID(w, id)
    case http.MethodPut:
        h.update(w, r, id)
    case http.MethodDelete:
        h.delete(w, id)
    default:
        helpers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
    }
}

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    products, err := h.service.FindAll(name)
    if err != nil {
        helpers.WriteError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    if products == nil {
        products = []Product{}
    }
    helpers.WriteJSON(w, http.StatusOK, toResponses(products))
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
    var req productRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    p := &Product{Name: req.Name, Price: req.Price, Stock: req.Stock, CategoryID: req.CategoryID}
    if err := h.service.Create(p); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusCreated, toResponse(*p))
}

func (h *ProductHandler) getByID(w http.ResponseWriter, id int) {
    p, err := h.service.FindByID(id)
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
    var req productRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    p := &Product{ID: id, Name: req.Name, Price: req.Price, Stock: req.Stock, CategoryID: req.CategoryID}
    if err := h.service.Update(p); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    helpers.WriteJSON(w, http.StatusOK, toResponse(*p))
}

func (h *ProductHandler) delete(w http.ResponseWriter, id int) {
    if err := h.service.Delete(id); err != nil {
        helpers.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
