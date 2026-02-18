package main

import (
	"database/sql"
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

func autoMigrate(db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			name VARCHAR(100) NOT NULL,
			role VARCHAR(20) DEFAULT 'cashier',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(64) PRIMARY KEY,
			user_id INT REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			log.Printf("Migration warning: %v", err)
		}
	}

	// Seed default admin if no users exist
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count == 0 {
		hash, err := service.HashPassword("admin123")
		if err != nil {
			log.Printf("Failed to hash seed password: %v", err)
			return
		}
		_, err = db.Exec(
			`INSERT INTO users (username, password_hash, name, role) VALUES ($1, $2, $3, $4)`,
			"admin", hash, "Administrator", "admin",
		)
		if err != nil {
			log.Printf("Failed to seed admin user: %v", err)
		} else {
			log.Println("Seeded default admin user (admin / admin123)")
		}
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

	autoMigrate(db)

	// Repositories
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Services
	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(productRepo, categoryRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	authService := service.NewAuthService(userRepo)

	// Handlers
	categoryHandler := handler.NewCategoryHandler(categoryService)
	productHandler := handler.NewProductHandler(productService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	authHandler := handler.NewAuthHandler(authService)

	// Middleware
	sessionAuth := middleware.SessionAuth(authService)

	mux := http.NewServeMux()

	// Auth routes (public)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("/api/auth/me", authHandler.HandleMe)

	// Public API routes
	mux.Handle("/api/products", http.HandlerFunc(productHandler.HandleProducts))
	mux.Handle("/categories", http.HandlerFunc(categoryHandler.HandleCategories))
	mux.Handle("/categories/", http.HandlerFunc(categoryHandler.HandleCategoryByID))
	mux.Handle("/api/report/hari-ini", http.HandlerFunc(transactionHandler.HandleTodayReport))
	mux.Handle("/api/report", http.HandlerFunc(transactionHandler.HandleReport))

	// Protected API routes (require session)
	mux.Handle("/api/products/", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), sessionAuth))
	mux.Handle("/api/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), sessionAuth))

	// Static pages
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/openapi.yaml", handler.OpenAPI)
	mux.HandleFunc("/docs", handler.Docs)

	// SPA
	mux.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app/", http.StatusTemporaryRedirect)
	})
	webFS := http.Dir("web")
	mux.HandleFunc("/app/", func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/app/")
		if p == "" {
			p = "index.html"
		}
		if _, err := os.Stat("web/" + p); os.IsNotExist(err) {
			http.ServeFile(w, r, "web/index.html")
			return
		}
		http.StripPrefix("/app/", http.FileServer(webFS)).ServeHTTP(w, r)
	})

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
