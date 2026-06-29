package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os/signal"
	"syscall"

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
	"go-kasir-api/internal/pkg/config"
	"go-kasir-api/internal/pkg/database"
	"go-kasir-api/internal/pkg/helpers"
	"go-kasir-api/internal/pkg/middleware"

	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	db, err := database.InitDB(cfg.DBConn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if cfg.AutoMigrate {
		if err := database.RunMigrations(db, cfg.MigrationsPath); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	}
	seedDefaultOrg(db, cfg.AdminPassword)

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

	authService := auth.NewAuthService(userRepo, branchRepo, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	categoryService := category.NewCategoryService(categoryRepo)
	productService := product.NewProductService(productRepo, categoryRepo)
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

	corsMiddleware := middleware.CORS(cfg.CORSAllowedOrigin)
	jwtMiddleware := middleware.JWTAuth(authService)
	adminOnly := middleware.RequireRole("admin")

	mux := http.NewServeMux()

	// Health & root
	mux.HandleFunc("/health", handlerHealth)
	mux.HandleFunc("/", handlerRoot)

	// Old API routes (backward compatibility)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/refresh", authHandler.HandleRefresh)
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
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.HandleRefresh)
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
		middleware.BodyLimit(cfg.MaxRequestBodyBytes),
		corsMiddleware,
		middleware.SecurityHeaders(),
		middleware.Logger(cfg.LogLevel),
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           wrapped,
		ReadTimeout:       cfg.ServerReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ServerShutdown)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := server.Close(); err != nil {
			log.Printf("server close failed: %v", err)
		}
	}
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/health", http.StatusFound)
}

func seedDefaultOrg(db *sql.DB, configuredAdminPass string) {
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
		adminPass := configuredAdminPass
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
