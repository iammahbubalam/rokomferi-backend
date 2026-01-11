package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/delivery/http/middleware"
	v1 "rokomferi-backend/internal/delivery/http/v1"
	"rokomferi-backend/internal/repository/postgres"
	"rokomferi-backend/internal/usecase"
	postgresPkg "rokomferi-backend/pkg/postgres"
	"rokomferi-backend/pkg/utils"
	"syscall"
	"time"
)

func main() {
	cfg := config.LoadConfig()
	utils.SetSecret(cfg.JWTSecret)

	// Initialize Database
	db, err := postgresPkg.NewClient(cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Verify connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL via NeonDB")

	// Set up Router
	mux := http.NewServeMux()

	// --- Modules Initialization ---

	// 2. Auth Module
	userRepo := postgres.NewUserRepository(db)
	authUC := usecase.NewAuthUsecase(
		userRepo,
		cfg.GoogleClientID,
		cfg.GoogleTokenInfoURL,
		cfg.AccessTokenExpiry,
		cfg.RefreshTokenExpiry,
	)
	authHandler := v1.NewAuthHandler(authUC)

	// 2. Catalog Module
	productRepo := postgres.NewProductRepository(db)
	catalogUC := usecase.NewCatalogUsecase(productRepo)
	catalogHandler := v1.NewCatalogHandler(catalogUC)

	// Admin Catalog Handlers
	adminCatalogHandler := v1.NewAdminCatalogHandler(catalogUC)

	// 3. Order Module
	orderRepo := postgres.NewOrderRepository(db)
	txManager := postgres.NewTransactionManager(db)
	orderUC := usecase.NewOrderUsecase(orderRepo, productRepo, txManager)
	orderHandler := v1.NewOrderHandler(orderUC)
	adminOrderHandler := v1.NewAdminOrderHandler(orderUC)

	// --- Routes ---

	// Auth
	mux.HandleFunc("POST /api/v1/auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh) // New
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/v1/auth/me", middleware.AuthMiddleware(http.HandlerFunc(authHandler.Me)))

	// User Profile / Address
	mux.Handle("POST /api/v1/user/addresses", middleware.AuthMiddleware(http.HandlerFunc(authHandler.AddAddress)))
	mux.Handle("GET /api/v1/user/addresses", middleware.AuthMiddleware(http.HandlerFunc(authHandler.GetAddresses)))

	// Catalog (Public)
	mux.HandleFunc("GET /api/v1/categories/tree", catalogHandler.GetCategories)
	mux.HandleFunc("GET /api/v1/products", catalogHandler.ListProducts)
	mux.HandleFunc("GET /api/v1/products/{slug}", catalogHandler.GetProductDetails)

	// Admin (Protected)
	adminMiddleware := func(h http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.AdminMiddleware(h))
	}

	mux.Handle("POST /api/v1/admin/products", adminMiddleware(adminCatalogHandler.CreateProduct))
	mux.Handle("PUT /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.UpdateProduct))
	mux.Handle("DELETE /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.DeleteProduct))
	mux.Handle("POST /api/v1/admin/inventory/adjust", adminMiddleware(adminCatalogHandler.AdjustStock))

	mux.Handle("GET /api/v1/admin/orders", adminMiddleware(adminOrderHandler.ListOrders))
	mux.Handle("PATCH /api/v1/admin/orders/{id}/status", adminMiddleware(adminOrderHandler.UpdateStatus))

	// Cart & Order (Protected)
	mux.Handle("GET /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetCart)))
	mux.Handle("POST /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.AddToCart)))
	mux.Handle("POST /api/v1/checkout", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.Checkout)))
	mux.Handle("GET /api/v1/orders", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetMyOrders)))

	// Health Check
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "db": "connected"}`))
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)

	// Apply CORS
	handler := middleware.CORS(mux)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Graceful Shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Printf("Server starting on %s", addr)

	// Wait for interrupt signal via channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
