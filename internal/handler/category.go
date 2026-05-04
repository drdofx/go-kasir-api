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

type CategoryHandler struct {
	service *service.CategoryService
}

func NewCategoryHandler(service *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetAll(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CategoryHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetAll(r.Context())
	if err != nil {
		log.Printf("GetAll error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, categories)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var category model.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Create(r.Context(), &category); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusCreated, category)
}

func (h *CategoryHandler) HandleCategoryByID(w http.ResponseWriter, r *http.Request) {
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

func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/categories/")
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	category, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			http.Error(w, "Category not found", http.StatusNotFound)
			return
		}
		log.Printf("GetByID error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, category)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/categories/")
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var category model.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	category.ID = id
	if err := h.service.Update(r.Context(), &category); err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			http.Error(w, "Category not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, category)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/categories/")
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			http.Error(w, "Category not found", http.StatusNotFound)
			return
		}
		log.Printf("Delete error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Category deleted successfully",
	})
}
