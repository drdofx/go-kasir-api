package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"go-kasir-api/internal/database"
	"go-kasir-api/internal/handler"
	"go-kasir-api/internal/middleware"
	"go-kasir-api/internal/repository"
	"go-kasir-api/internal/service"

	"github.com/spf13/viper"
)

type Config struct {
	Port              string `mapstructure:"PORT"`
	DBConn            string `mapstructure:"DB_CONN"`
	APIKey            string `mapstructure:"API_KEY"`
	CORSAllowedOrigin string `mapstructure:"CORS_ALLOWED_ORIGIN"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`
}

func loadConfig() Config {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigFile(".env")
		_ = viper.ReadInConfig()
	}

	return Config{
		Port:              viper.GetString("PORT"),
		DBConn:            viper.GetString("DB_CONN"),
		APIKey:            viper.GetString("API_KEY"),
		CORSAllowedOrigin: viper.GetString("CORS_ALLOWED_ORIGIN"),
		LogLevel:          viper.GetString("LOG_LEVEL"),
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
	if config.APIKey == "" {
		log.Fatal("API_KEY is required")
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

	mux := http.NewServeMux()
	apiKey := middleware.APIKey(config.APIKey)

	mux.Handle("/api/products", http.HandlerFunc(productHandler.HandleProducts))
	mux.Handle("/api/products/", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), apiKey))
	mux.Handle("/categories", http.HandlerFunc(categoryHandler.HandleCategories))
	mux.Handle("/categories/", http.HandlerFunc(categoryHandler.HandleCategoryByID))
	mux.Handle("/api/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), apiKey))
	mux.Handle("/api/report/hari-ini", http.HandlerFunc(transactionHandler.HandleTodayReport))
	mux.Handle("/api/report", http.HandlerFunc(transactionHandler.HandleReport))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/openapi.yaml", handler.OpenAPI)
	mux.HandleFunc("/docs", handler.Docs)

	rootHandler := middleware.Chain(
		mux,
		middleware.RequestID(),
		middleware.CORS(config.CORSAllowedOrigin),
		middleware.Logger(config.LogLevel),
	)

	addr := "0.0.0.0:" + config.Port
	fmt.Println("Server running at", addr)
	if err := http.ListenAndServe(addr, rootHandler); err != nil {
		fmt.Println("failed to start server", err)
	}
}
