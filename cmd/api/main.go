package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-kasir-api/internal/domain/auth"
	"go-kasir-api/internal/domain/category"
	"go-kasir-api/internal/domain/customer"
	"go-kasir-api/internal/domain/inventory"
	"go-kasir-api/internal/domain/payment"
	"go-kasir-api/internal/domain/product"
	"go-kasir-api/internal/domain/purchaseorder"
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
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

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

	if err := database.RunMigrations(db, viper.GetString("MIGRATIONS_PATH")); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

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

	catRepoForProduct := &categoryRepoWrapper{categoryRepo}

	authService := auth.NewAuthService(userRepo, viper.GetString("JWT_SECRET"))
	categoryService := category.NewCategoryService(categoryRepo)
	productService := product.NewProductService(productRepo, catRepoForProduct)
	customerService := customer.NewCustomerService(customerRepo)
	paymentService := payment.NewPaymentService(paymentRepo)
	returnService := returns.NewReturnService(returnRepo, transactionRepo)
	supplierService := supplier.NewSupplierService(supplierRepo)
	poService := purchaseorder.NewPOService(purchaseOrderRepo, supplierRepo)
	inventoryService := inventory.NewInventoryService(inventoryRepo)
	transactionService := transaction.NewTransactionService(transactionRepo, customerRepo, paymentService)
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
	transactionHandler := transaction.NewTransactionHandler(transactionService)
	reportHandler := report.NewReportHandler(reportService)

	corsMiddleware := middleware.CORS(viper.GetString("CORS_ALLOWED_ORIGIN"))
	jwtMiddleware := middleware.JWTAuth(authService)

	mux := http.NewServeMux()

	// Health & root
	mux.HandleFunc("/health", handlerHealth)
	mux.HandleFunc("/", handlerRoot)

	// Old API routes (backward compatibility)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("/api/auth/me", authHandler.HandleMe)
	mux.Handle("/api/auth/change-password", middleware.Chain(http.HandlerFunc(authHandler.HandleChangePassword), jwtMiddleware))
	mux.Handle("/api/products", middleware.Chain(http.HandlerFunc(productHandler.HandleProducts), jwtMiddleware))
	mux.Handle("/api/products/", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), jwtMiddleware))
	mux.Handle("/api/categories", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategories), jwtMiddleware))
	mux.Handle("/api/categories/", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategoryByID), jwtMiddleware))
	mux.Handle("/api/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), jwtMiddleware))
	mux.Handle("/api/transactions", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTransactions), jwtMiddleware))
	mux.Handle("/api/report/hari-ini", middleware.Chain(http.HandlerFunc(reportHandler.HandleTodayReport), jwtMiddleware))
	mux.Handle("/api/report", middleware.Chain(http.HandlerFunc(reportHandler.HandleReport), jwtMiddleware))

	// V1 API routes
	mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("/api/v1/auth/me", authHandler.HandleMe)
	mux.Handle("/api/v1/auth/change-password", middleware.Chain(http.HandlerFunc(authHandler.HandleChangePassword), jwtMiddleware))
	mux.Handle("/api/v1/products", middleware.Chain(http.HandlerFunc(productHandler.HandleProducts), jwtMiddleware))
	mux.Handle("/api/v1/products/", middleware.Chain(http.HandlerFunc(productHandler.HandleProductByID), jwtMiddleware))
	mux.Handle("/api/v1/categories", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategories), jwtMiddleware))
	mux.Handle("/api/v1/categories/", middleware.Chain(http.HandlerFunc(categoryHandler.HandleCategoryByID), jwtMiddleware))
	mux.Handle("/api/v1/checkout", middleware.Chain(http.HandlerFunc(transactionHandler.HandleCheckout), jwtMiddleware))
	mux.Handle("/api/v1/transactions", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTransactions), jwtMiddleware))
	mux.Handle("/api/v1/transactions/", middleware.Chain(http.HandlerFunc(transactionHandler.HandleTransactionByID), jwtMiddleware))
	mux.Handle("/api/v1/customers", middleware.Chain(http.HandlerFunc(customerHandler.HandleCustomers), jwtMiddleware))
	mux.Handle("/api/v1/customers/", middleware.Chain(http.HandlerFunc(customerHandler.HandleCustomerByID), jwtMiddleware))
	mux.Handle("/api/v1/payment-types", middleware.Chain(http.HandlerFunc(paymentHandler.HandlePaymentTypes), jwtMiddleware))
	mux.Handle("/api/v1/returns", middleware.Chain(http.HandlerFunc(returnHandler.HandleReturns), jwtMiddleware))
	mux.Handle("/api/v1/returns/", middleware.Chain(http.HandlerFunc(returnHandler.HandleGetReturn), jwtMiddleware))
	mux.Handle("/api/v1/suppliers", middleware.Chain(http.HandlerFunc(supplierHandler.HandleSuppliers), jwtMiddleware))
	mux.Handle("/api/v1/suppliers/", middleware.Chain(http.HandlerFunc(supplierHandler.HandleSupplierByID), jwtMiddleware))
	mux.Handle("/api/v1/purchase-orders", middleware.Chain(http.HandlerFunc(poHandler.HandlePOs), jwtMiddleware))
	mux.Handle("/api/v1/purchase-orders/", middleware.Chain(http.HandlerFunc(poHandler.HandlePOByID), jwtMiddleware))
	mux.Handle("/api/v1/inventory/alerts", middleware.Chain(http.HandlerFunc(inventoryHandler.HandleAlerts), jwtMiddleware))
	mux.Handle("/api/v1/inventory/alerts/", middleware.Chain(http.HandlerFunc(inventoryHandler.HandleSetThreshold), jwtMiddleware))
	mux.Handle("/api/v1/users", middleware.Chain(http.HandlerFunc(authHandler.HandleUsers), jwtMiddleware))
	mux.Handle("/api/v1/users/", middleware.Chain(http.HandlerFunc(authHandler.HandleUpdateUserRole), jwtMiddleware))
	mux.Handle("/api/v1/report/hari-ini", middleware.Chain(http.HandlerFunc(reportHandler.HandleTodayReport), jwtMiddleware))
	mux.Handle("/api/v1/report", middleware.Chain(http.HandlerFunc(reportHandler.HandleReport), jwtMiddleware))
	mux.Handle("/api/v1/report/dashboard", middleware.Chain(http.HandlerFunc(reportHandler.HandleDashboard), jwtMiddleware))
	mux.Handle("/api/v1/report/weekly", middleware.Chain(http.HandlerFunc(reportHandler.HandleWeeklyReport), jwtMiddleware))
	mux.Handle("/api/v1/report/monthly", middleware.Chain(http.HandlerFunc(reportHandler.HandleMonthlyReport), jwtMiddleware))
	mux.Handle("/api/v1/report/by-category", middleware.Chain(http.HandlerFunc(reportHandler.HandleSalesByCategory), jwtMiddleware))
	mux.Handle("/api/v1/report/by-product", middleware.Chain(http.HandlerFunc(reportHandler.HandleSalesByProduct), jwtMiddleware))
	mux.Handle("/api/v1/report/export", middleware.Chain(http.HandlerFunc(reportHandler.HandleExportCSV), jwtMiddleware))

	wrapped := middleware.Chain(mux,
		middleware.RequestID(),
		corsMiddleware,
		middleware.SecurityHeaders(),
		middleware.Logger(viper.GetString("LOG_LEVEL")),
	)

	server := &http.Server{
		Addr:    ":" + viper.GetString("PORT"),
		Handler: wrapped,
	}

	go func() {
		log.Printf("server starting on port %s", viper.GetString("PORT"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")
	server.Close()
}

type categoryRepoWrapper struct {
	repo category.CategoryRepository
}

func (w *categoryRepoWrapper) FindByID(id int) (*product.Category, error) {
	c, err := w.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &product.Category{ID: c.ID, Name: c.Name, Description: c.Description}, nil
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/health", http.StatusFound)
}
