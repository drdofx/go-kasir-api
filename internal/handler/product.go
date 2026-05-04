package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-kasir-api/internal/model"
	"go-kasir-api/internal/repository"
	"go-kasir-api/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetAll(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProductHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	products, err := h.service.GetAll(r.Context(), name)
	if err != nil {
		log.Printf("GetAll error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, products)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var product model.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Create(r.Context(), &product); err != nil {
		if errors.Is(err, service.ErrInvalidCategoryID) {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}
		if errors.Is(err, repository.ErrCategoryNotFound) {
			http.Error(w, "Category not found", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusCreated, product)
}

func (h *ProductHandler) HandleProductByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetByID(w, r)
	case http.MethodPut:
		h.Update(w, r)
	case http.MethodDelete:
		h.Delete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		log.Printf("GetByID error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var product model.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	product.ID = id
	if err := h.service.Update(r.Context(), &product); err != nil {
		if errors.Is(err, service.ErrInvalidCategoryID) {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}
		if errors.Is(err, repository.ErrCategoryNotFound) {
			http.Error(w, "Category not found", http.StatusBadRequest)
			return
		}
		if errors.Is(err, repository.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		log.Printf("Delete error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Product deleted successfully",
	})
}
