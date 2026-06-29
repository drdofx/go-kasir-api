package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go-kasir-api/internal/domain/auth"
	"go-kasir-api/internal/domain/branch"
	"go-kasir-api/internal/domain/category"
	"go-kasir-api/internal/domain/customer"
	"go-kasir-api/internal/domain/inventory"
	"go-kasir-api/internal/domain/payment"
	"go-kasir-api/internal/domain/product"
	"go-kasir-api/internal/domain/purchaseorder"
	"go-kasir-api/internal/domain/receipt"
	"go-kasir-api/internal/domain/report"
	"go-kasir-api/internal/domain/returns"
	"go-kasir-api/internal/domain/supplier"
	"go-kasir-api/internal/domain/transaction"
	"go-kasir-api/internal/pkg/database"
	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func main() {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("CORS_ALLOWED_ORIGIN", "http://localhost:8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MIGRATIONS_PATH", "migrations")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("AUTO_MIGRATE", true)
	viper.SetDefault("SERVER_READ_TIMEOUT", "10s")
	viper.SetDefault("SERVER_READ_HEADER_TIMEOUT", "5s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "30s")
	viper.SetDefault("SERVER_IDLE_TIMEOUT", "120s")
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "10s")
	viper.SetDefault("MAX_REQUEST_BODY_BYTES", 1048576)
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !os.IsNotExist(err) && !errors.As(err, &notFound) {
			log.Fatalf("failed to read config: %v", err)
		}
	}

	if err := validateConfig(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	logLevel, err := zerolog.ParseLevel(viper.GetString("LOG_LEVEL"))
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	db, err := database.InitDB(viper.GetString("DB_CONN"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if viper.GetBool("AUTO_MIGRATE") {
		if err := database.RunMigrations(db, viper.GetString("MIGRATIONS_PATH")); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	}
	seedDefaultOrg(db)

	userRepo := auth.NewUserRepository(db)
	productRepo := product.NewProductRepository(db)
	categoryRepo := category.NewCategoryRepository(db)
	transactionRepo := transaction.NewTransactionRepository(db)
	customerRepo := customer.NewCustomerRepository(db)
	paymentRepo := payment.NewPaymentRepository(db)
	returnRepo := returns.NewReturnRepository(db)
	supplierRepo := supplier.NewSupplierRepository(db)
	purchaseOrderRepo := purchaseorder.NewPurchaseOrderRepository(db)
	inventoryRepo := inventory.NewInventoryRepository(db)
	receiptRepo := receipt.NewReceiptRepository(db)
	branchRepo := branch.NewBranchRepository(db)

	catRepoForProduct := &categoryRepoWrapper{categoryRepo}
	branchRepoForAuth := &authBranchRepoWrapper{branchRepo}

	authService := auth.NewAuthService(userRepo, branchRepoForAuth, viper.GetString("JWT_SECRET"))
	categoryService := category.NewCategoryService(categoryRepo)
	productService := product.NewProductService(productRepo, catRepoForProduct)
	customerService := customer.NewCustomerService(customerRepo)
	paymentService := payment.NewPaymentService(paymentRepo)
	returnService := returns.NewReturnService(returnRepo, transactionRepo)
	supplierService := supplier.NewSupplierService(supplierRepo)
	poService := purchaseorder.NewPOService(purchaseOrderRepo, supplierRepo)
	inventoryService := inventory.NewInventoryService(inventoryRepo)
	branchService := branch.NewBranchService(branchRepo)
	transactionService := transaction.NewTransactionService(transactionRepo, customerRepo, paymentService, receiptRepo)
	reportService := report.NewReportService(db)

	authHandler := auth.NewAuthHandler(authService)
	productHandler := product.NewProductHandler(productService)
	categoryHandler := category.NewCategoryHandler(categoryService)
	customerHandler := customer.NewCustomerHandler(customerService)
	paymentHandler := payment.NewPaymentHandler(paymentService)
	returnHandler := returns.NewReturnHandler(returnService)
	supplierHandler := supplier.NewSupplierHandler(supplierService)
	poHandler := purchaseorder.NewPOHandler(poService)
	inventoryHandler := inventory.NewInventoryHandler(inventoryService)
	receiptHandler := receipt.NewReceiptHandler(receipt.NewReceiptService(receiptRepo))
	branchHandler := branch.NewBranchHandler(branchService)
	transactionHandler := transaction.NewTransactionHandler(transactionService)
	reportHandler := report.NewReportHandler(reportService)

	corsMiddleware := middleware.CORS(viper.GetString("CORS_ALLOWED_ORIGIN"))
	jwtMiddleware := middleware.JWTAuth(authService)
	adminOnly := middleware.RequireRole("admin")

	mux := http.NewServeMux()

	// Health & root
	mux.HandleFunc("/health", handlerHealth)
	mux.HandleFunc("/", handlerRoot)

	// Old API routes (backward compatibility)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.Handle("/api/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.HandleLogout), jwtMiddleware))
	mux.Handle("/api/auth/me", middleware.Chain(http.HandlerFunc(authHandler.HandleMe), jwtMiddleware))
	mux.Handle("/api/auth/change-password", middleware.Chain(http.HandlerFunc(authHandler.HandleChangePassword), jwtMiddleware))
	mux.Handle("/api/products", middleware.Chain(http.HandlerFunc(productHandler.HandleProducts), jwtMiddleware))
	mux.Handle("/api/products/{id}", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), jwtMiddleware))
	mux.Handle("/api/categories", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategories), jwtMiddleware))
	mux.Handle("/api/categories/{id}", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategoryByID), jwtMiddleware))
	mux.Handle("/api/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), jwtMiddleware))
	mux.Handle("/api/transactions", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTransactions), jwtMiddleware))
	mux.Handle("/api/report/hari-ini", middleware.Chain(http.HandlerFunc(reportHandler.HandleTodayReport), jwtMiddleware))
	mux.Handle("/api/report", middleware.Chain(http.HandlerFunc(reportHandler.HandleReport), jwtMiddleware))

	// V1 API routes
	mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.HandleLogout), jwtMiddleware))
	mux.Handle("/api/v1/auth/me", middleware.Chain(http.HandlerFunc(authHandler.HandleMe), jwtMiddleware))
	mux.Handle("/api/v1/auth/change-password", middleware.Chain(http.HandlerFunc(authHandler.HandleChangePassword), jwtMiddleware))
	mux.Handle("/api/v1/products", middleware.Chain(http.HandlerFunc(productHandler.HandleProducts), jwtMiddleware))
	mux.Handle("/api/v1/products/{id}", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), jwtMiddleware))
	mux.Handle("/api/v1/categories", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategories), jwtMiddleware))
	mux.Handle("/api/v1/categories/{id}", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategoryByID), jwtMiddleware))
	mux.Handle("/api/v1/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), jwtMiddleware))
	mux.Handle("/api/v1/transactions", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTransactions), jwtMiddleware))
	mux.Handle("/api/v1/transactions/{id}", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTransactionByID), jwtMiddleware))
	mux.Handle("/api/v1/customers", middleware.Chain(http.HandlerFunc(customerHandler.HandleCustomers), jwtMiddleware))
	mux.Handle("/api/v1/customers/{id}", middleware.Chain(http.HandlerFunc(customerHandler.HandleCustomerByID), jwtMiddleware))
	mux.Handle("/api/v1/customers/{id}/purchases", middleware.Chain(http.HandlerFunc(customerHandler.HandleCustomerHistory), jwtMiddleware))
	mux.Handle("/api/v1/customers/{id}/history", middleware.Chain(http.HandlerFunc(customerHandler.HandleCustomerHistory), jwtMiddleware))
	mux.Handle("/api/v1/payment-types", middleware.Chain(http.HandlerFunc(paymentHandler.HandlePaymentTypes), jwtMiddleware))
	mux.Handle("/api/v1/returns", middleware.Chain(http.HandlerFunc(returnHandler.HandleReturns), jwtMiddleware))
	mux.Handle("/api/v1/returns/{id}", middleware.Chain(http.HandlerFunc(returnHandler.HandleGetReturn), jwtMiddleware))
	mux.Handle("/api/v1/suppliers", middleware.Chain(http.HandlerFunc(supplierHandler.HandleSuppliers), jwtMiddleware))
	mux.Handle("/api/v1/suppliers/{id}", middleware.Chain(http.HandlerFunc(supplierHandler.HandleSupplierByID), jwtMiddleware))
	mux.Handle("/api/v1/purchase-orders", middleware.Chain(http.HandlerFunc(poHandler.HandlePOs), jwtMiddleware))
	mux.Handle("/api/v1/purchase-orders/{id}", middleware.Chain(http.HandlerFunc(poHandler.HandlePOByID), jwtMiddleware))
	mux.Handle("/api/v1/purchase-orders/{id}/receive", middleware.Chain(http.HandlerFunc(poHandler.HandleReceivePO), jwtMiddleware))
	mux.Handle("/api/v1/inventory/alerts", middleware.Chain(http.HandlerFunc(inventoryHandler.HandleAlerts), jwtMiddleware))
	mux.Handle("/api/v1/inventory/alerts/{id}", middleware.Chain(http.HandlerFunc(inventoryHandler.HandleSetThreshold), jwtMiddleware))
	mux.Handle("/api/v1/users", middleware.Chain(http.HandlerFunc(authHandler.HandleUsers), jwtMiddleware, adminOnly))
	mux.Handle("/api/v1/users/{id}", middleware.Chain(http.HandlerFunc(authHandler.HandleUpdateUserRole), jwtMiddleware, adminOnly))
	mux.Handle("/api/v1/receipts/{id}", middleware.Chain(http.HandlerFunc(receiptHandler.HandleGetReceipt), jwtMiddleware))
	mux.Handle("/api/v1/auth/switch-branch", middleware.Chain(http.HandlerFunc(authHandler.HandleSwitchBranch), jwtMiddleware))
	mux.Handle("/api/v1/branches", middleware.Chain(http.HandlerFunc(branchHandler.HandleBranches), jwtMiddleware, adminOnly))
	mux.Handle("/api/v1/branches/{id}", middleware.Chain(http.HandlerFunc(branchHandler.HandleBranchByID), jwtMiddleware, adminOnly))
	mux.Handle("/api/v1/report/hari-ini", middleware.Chain(http.HandlerFunc(reportHandler.HandleTodayReport), jwtMiddleware))
	mux.Handle("/api/v1/report/today", middleware.Chain(http.HandlerFunc(reportHandler.HandleTodayReport), jwtMiddleware))
	mux.Handle("/api/v1/report", middleware.Chain(http.HandlerFunc(reportHandler.HandleReport), jwtMiddleware))
	mux.Handle("/api/v1/report/dashboard", middleware.Chain(http.HandlerFunc(reportHandler.HandleDashboard), jwtMiddleware))
	mux.Handle("/api/v1/report/weekly", middleware.Chain(http.HandlerFunc(reportHandler.HandleWeeklyReport), jwtMiddleware))
	mux.Handle("/api/v1/report/monthly", middleware.Chain(http.HandlerFunc(reportHandler.HandleMonthlyReport), jwtMiddleware))
	mux.Handle("/api/v1/report/by-category", middleware.Chain(http.HandlerFunc(reportHandler.HandleSalesByCategory), jwtMiddleware))
	mux.Handle("/api/v1/report/by-product", middleware.Chain(http.HandlerFunc(reportHandler.HandleSalesByProduct), jwtMiddleware))
	mux.Handle("/api/v1/report/export", middleware.Chain(http.HandlerFunc(reportHandler.HandleExportCSV), jwtMiddleware))

	wrapped := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.BodyLimit(viper.GetInt64("MAX_REQUEST_BODY_BYTES")),
		corsMiddleware,
		middleware.SecurityHeaders(),
		middleware.Logger(viper.GetString("LOG_LEVEL")),
	)

	server := &http.Server{
		Addr:              ":" + viper.GetString("PORT"),
		Handler:           wrapped,
		ReadTimeout:       mustDuration("SERVER_READ_TIMEOUT"),
		ReadHeaderTimeout: mustDuration("SERVER_READ_HEADER_TIMEOUT"),
		WriteTimeout:      mustDuration("SERVER_WRITE_TIMEOUT"),
		IdleTimeout:       mustDuration("SERVER_IDLE_TIMEOUT"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server starting on port %s", viper.GetString("PORT"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), mustDuration("SERVER_SHUTDOWN_TIMEOUT"))
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := server.Close(); err != nil {
			log.Printf("server close failed: %v", err)
		}
	}
}

func validateConfig() error {
	if strings.TrimSpace(viper.GetString("DB_CONN")) == "" {
		return fmt.Errorf("DB_CONN is required")
	}

	jwtSecret := strings.TrimSpace(viper.GetString("JWT_SECRET"))
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	appEnv := strings.ToLower(strings.TrimSpace(viper.GetString("APP_ENV")))
	if appEnv == "production" {
		if len(jwtSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
		}
		if jwtSecret == "change-this-secret-in-production" {
			return fmt.Errorf("JWT_SECRET must not use the default development value in production")
		}
		if strings.TrimSpace(viper.GetString("ADMIN_PASSWORD")) == "" {
			return fmt.Errorf("ADMIN_PASSWORD is required in production")
		}
	}
	if viper.GetInt64("MAX_REQUEST_BODY_BYTES") < 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES must be greater than or equal to 0")
	}

	for _, key := range []string{
		"SERVER_READ_TIMEOUT",
		"SERVER_READ_HEADER_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
		"SERVER_SHUTDOWN_TIMEOUT",
	} {
		if _, err := time.ParseDuration(viper.GetString(key)); err != nil {
			return fmt.Errorf("%s must be a valid duration: %w", key, err)
		}
	}

	return nil
}

func mustDuration(key string) time.Duration {
	d, err := time.ParseDuration(viper.GetString(key))
	if err != nil {
		log.Fatalf("invalid duration for %s: %v", key, err)
	}
	return d
}

type categoryRepoWrapper struct {
	repo category.CategoryRepository
}

func (w *categoryRepoWrapper) FindByIDForOrg(orgID, id int) (*product.Category, error) {
	c, err := w.repo.FindByIDForOrg(orgID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &product.Category{ID: c.ID, Name: c.Name, Description: c.Description}, nil
}

type authBranchRepoWrapper struct {
	repo branch.BranchRepository
}

func (w *authBranchRepoWrapper) FindByID(id int) (*auth.Branch, error) {
	b, err := w.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	return &auth.Branch{ID: b.ID, OrganizationID: b.OrganizationID}, nil
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/health", http.StatusFound)
}

func seedDefaultOrg(db *sql.DB) {
	var orgID int
	err := db.QueryRow("SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&orgID)
	if err == sql.ErrNoRows {
		err = db.QueryRow("INSERT INTO organizations (name, slug) VALUES ('Default Organization', 'default') RETURNING id").Scan(&orgID)
		if err != nil {
			log.Printf("seed org: %v", err)
			return
		}
		log.Printf("created default org (id=%d)", orgID)
	}
	var branchID int
	err = db.QueryRow("SELECT id FROM branches ORDER BY id LIMIT 1").Scan(&branchID)
	if err == sql.ErrNoRows {
		err = db.QueryRow("INSERT INTO branches (organization_id, name, code) VALUES ($1, 'Main Branch', 'main') RETURNING id", orgID).Scan(&branchID)
		if err != nil {
			log.Printf("seed branch: %v", err)
			return
		}
		log.Printf("created default branch (id=%d) for org %d", branchID, orgID)
	}
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		adminPass := viper.GetString("ADMIN_PASSWORD")
		if adminPass == "" {
			adminPass = "kasir123"
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("seed user hash: %v", err)
			return
		}
		_, err = db.Exec("INSERT INTO users (username, password_hash, name, role, organization_id, branch_id) VALUES ($1, $2, $3, $4, $5, $6)",
			"kasir", string(hash), "Kasir Utama", "admin", orgID, branchID)
		if err != nil {
			log.Printf("seed user: %v", err)
			return
		}
		log.Printf("seeded admin user: kasir (org=%d, branch=%d)", orgID, branchID)
	} else {
		db.Exec("UPDATE users SET organization_id = COALESCE(organization_id, $1), branch_id = COALESCE(branch_id, $2) WHERE organization_id IS NULL OR organization_id = 0", orgID, branchID)
	}
	db.Exec("UPDATE categories SET organization_id = $1 WHERE organization_id IS NULL OR organization_id = 0", orgID)
	db.Exec("UPDATE products SET organization_id = $1 WHERE organization_id IS NULL OR organization_id = 0", orgID)
	db.Exec("UPDATE transactions SET organization_id = COALESCE(organization_id, $1), branch_id = COALESCE(branch_id, $2) WHERE organization_id IS NULL OR organization_id = 0", orgID, branchID)
	db.Exec("UPDATE customers SET organization_id = $1 WHERE organization_id IS NULL OR organization_id = 0", orgID)
	db.Exec("UPDATE suppliers SET organization_id = $1 WHERE organization_id IS NULL OR organization_id = 0", orgID)
	db.Exec("UPDATE purchase_orders SET organization_id = COALESCE(organization_id, $1), branch_id = COALESCE(branch_id, $2) WHERE organization_id IS NULL OR organization_id = 0", orgID, branchID)
	db.Exec("UPDATE returns SET organization_id = $1 WHERE organization_id IS NULL OR organization_id = 0", orgID)
	db.Exec(`INSERT INTO product_stocks (product_id, branch_id, stock) SELECT id, $1, 0 FROM products WHERE id NOT IN (SELECT product_id FROM product_stocks WHERE branch_id = $1)`, branchID)
}
