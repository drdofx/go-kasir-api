package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"go-kasir-api/internal/database"
	"go-kasir-api/internal/handler"
	"go-kasir-api/internal/repository"
	"go-kasir-api/internal/service"

	"github.com/spf13/viper"
)

type Config struct {
	Port   string `mapstructure:"PORT"`
	DBConn string `mapstructure:"DB_CONN"`
}

func loadConfig() Config {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigFile(".env")
		_ = viper.ReadInConfig()
	}

	return Config{
		Port:   viper.GetString("PORT"),
		DBConn: viper.GetString("DB_CONN"),
	}
}

func main() {
	config := loadConfig()
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.DBConn == "" {
		log.Fatal("DB_CONN is required")
	}

	db, err := database.InitDB(config.DBConn)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categoryService)

	productRepo := repository.NewProductRepository(db)
	productService := service.NewProductService(productRepo, categoryRepo)
	productHandler := handler.NewProductHandler(productService)

	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(transactionRepo)
	transactionHandler := handler.NewTransactionHandler(transactionService)

	http.HandleFunc("/api/products", productHandler.HandleProducts)
	http.HandleFunc("/api/products/", productHandler.HandleProductByID)
	http.HandleFunc("/categories", categoryHandler.HandleCategories)
	http.HandleFunc("/categories/", categoryHandler.HandleCategoryByID)
	http.HandleFunc("/api/checkout", transactionHandler.HandleCheckout)
	http.HandleFunc("/api/report/hari-ini", transactionHandler.HandleTodayReport)
	http.HandleFunc("/api/report", transactionHandler.HandleReport)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health", http.StatusTemporaryRedirect)
	})
	http.HandleFunc("/health", handler.Health)
	http.HandleFunc("/openapi.yaml", handler.OpenAPI)
	http.HandleFunc("/docs", handler.Docs)

	addr := "0.0.0.0:" + config.Port
	fmt.Println("Server running at", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println("failed to start server", err)
	}
}
