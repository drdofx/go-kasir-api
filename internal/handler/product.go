package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go-kasir-api/internal/data"
	"go-kasir-api/internal/model"
)

// ProductCollection handles GET and POST for /api/products.
func ProductCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data.Products)
	case http.MethodPost:
		var productNew model.Product
		if err := json.NewDecoder(r.Body).Decode(&productNew); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		productNew.ID = len(data.Products) + 1
		data.Products = append(data.Products, productNew)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(productNew)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// ProductItem handles GET, PUT, and DELETE for /api/products/{id}.
func ProductItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getProductByID(w, r)
	case http.MethodPut:
		updateProduct(w, r)
	case http.MethodDelete:
		deleteProduct(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func getProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/products/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Product ID", http.StatusBadRequest)
		return
	}

	for _, p := range data.Products {
		if p.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
	}

	http.Error(w, "Product not found", http.StatusNotFound)
}

func updateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/products/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Product ID", http.StatusBadRequest)
		return
	}

	var productUpdate model.Product
	if err := json.NewDecoder(r.Body).Decode(&productUpdate); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	for i := range data.Products {
		if data.Products[i].ID == id {
			productUpdate.ID = id
			data.Products[i] = productUpdate

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(productUpdate)
			return
		}
	}

	http.Error(w, "Product not found", http.StatusNotFound)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/products/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Product ID", http.StatusBadRequest)
		return
	}

	for i, p := range data.Products {
		if p.ID == id {
			data.Products = append(data.Products[:i], data.Products[i+1:]...)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "delete success",
			})
			return
		}
	}

	http.Error(w, "Product not found", http.StatusNotFound)
}
