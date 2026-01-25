package main

import (
	"fmt"
	"net/http"

	"go-kasir-api/internal/handler"
)

func main() {
	http.HandleFunc("/api/products/", handler.ProductItem)
	http.HandleFunc("/api/products", handler.ProductCollection)
	http.HandleFunc("/categories/", handler.CategoryItem)
	http.HandleFunc("/categories", handler.CategoryCollection)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health", http.StatusTemporaryRedirect)
	})
	http.HandleFunc("/health", handler.Health)
	http.HandleFunc("/openapi.yaml", handler.OpenAPI)
	http.HandleFunc("/docs", handler.Docs)

	fmt.Println("Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("failed to start server")
	}
}
