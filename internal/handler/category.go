package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go-kasir-api/internal/data"
	"go-kasir-api/internal/model"
)

// CategoryCollection handles GET and POST for /categories.
func CategoryCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Categories)
	case http.MethodPost:
		var categoryNew model.Category
		if err := json.NewDecoder(r.Body).Decode(&categoryNew); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		categoryNew.ID = len(data.Categories) + 1
		data.Categories = append(data.Categories, categoryNew)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(categoryNew)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// CategoryItem handles GET, PUT, and DELETE for /categories/{id}.
func CategoryItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getCategoryByID(w, r)
	case http.MethodPut:
		updateCategory(w, r)
	case http.MethodDelete:
		deleteCategory(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func getCategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Category ID", http.StatusBadRequest)
		return
	}

	for _, c := range data.Categories {
		if c.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(c)
			return
		}
	}

	http.Error(w, "Category not found", http.StatusNotFound)
}

func updateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Category ID", http.StatusBadRequest)
		return
	}

	var categoryUpdate model.Category
	if err := json.NewDecoder(r.Body).Decode(&categoryUpdate); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	for i := range data.Categories {
		if data.Categories[i].ID == id {
			categoryUpdate.ID = id
			data.Categories[i] = categoryUpdate

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(categoryUpdate)
			return
		}
	}

	http.Error(w, "Category not found", http.StatusNotFound)
}

func deleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Category ID", http.StatusBadRequest)
		return
	}

	for i, c := range data.Categories {
		if c.ID == id {
			data.Categories = append(data.Categories[:i], data.Categories[i+1:]...)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "delete success",
			})
			return
		}
	}

	http.Error(w, "Category not found", http.StatusNotFound)
}
