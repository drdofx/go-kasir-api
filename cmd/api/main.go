package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	CORSAllowedOrigin string `mapstructure:"CORS_ALLOWED_ORIGIN"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`
	AdminPassword     string `mapstructure:"ADMIN_PASSWORD"`
	MigrationsPath    string `mapstructure:"MIGRATIONS_PATH"`
}

func loadConfig() Config {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigFile(".env")
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: failed to read .env file: %v", err)
		}
	}

	return Config{
		Port:              viper.GetString("PORT"),
		DBConn:            viper.GetString("DB_CONN"),
		CORSAllowedOrigin: viper.GetString("CORS_ALLOWED_ORIGIN"),
		LogLevel:          viper.GetString("LOG_LEVEL"),
		AdminPassword:     viper.GetString("ADMIN_PASSWORD"),
		MigrationsPath:    viper.GetString("MIGRATIONS_PATH"),
	}
}

func seedAdmin(db *sql.DB, password string) {
	if password == "" {
		log.Println("Skipping admin seed: ADMIN_PASSWORD not set")
		return
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		log.Printf("Warning: failed to check user count: %v", err)
		return
	}
	if count > 0 {
		return
	}

	hash, err := service.HashPassword(password)
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
		log.Println("Seeded default admin user")
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

	if err := database.RunMigrations(db, config.MigrationsPath); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	seedAdmin(db, config.AdminPassword)

	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	userRepo := repository.NewUserRepository(db)

	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(productRepo, categoryRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	authService := service.NewAuthService(userRepo)

	categoryHandler := handler.NewCategoryHandler(categoryService)
	productHandler := handler.NewProductHandler(productService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	authHandler := handler.NewAuthHandler(authService)

	sessionAuth := middleware.SessionAuth(authService)

	mux := http.NewServeMux()

	// Auth routes (public)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("/api/auth/me", authHandler.HandleMe)

	// Protected API routes (require session)
	mux.Handle("/api/products", middleware.Chain(http.HandlerFunc(productHandler.HandleProducts), sessionAuth))
	mux.Handle("/api/products/", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), sessionAuth))
	mux.Handle("/api/categories", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategories), sessionAuth))
	mux.Handle("/api/categories/", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategoryByID), sessionAuth))
	mux.Handle("/api/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), sessionAuth))
	mux.Handle("/api/report/hari-ini", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTodayReport), sessionAuth))
	mux.Handle("/api/report", middleware.Chain(http.HandlerFunc(transactionHandler.HandleReport), sessionAuth))

	// Static pages
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
		middleware.SecurityHeaders(),
		middleware.Logger(config.LogLevel),
	)

	addr := "0.0.0.0:" + config.Port

	srv := &http.Server{
		Addr:         addr,
		Handler:      rootHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Println("Server running at", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-done
	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
