package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/delivery/http/middleware"
	v1 "rokomferi-backend/internal/delivery/http/v1"
	sqlcrepo "rokomferi-backend/internal/repository/sqlc"
	"rokomferi-backend/internal/usecase"
	"rokomferi-backend/pkg/logger"
	"rokomferi-backend/pkg/storage"
	"rokomferi-backend/pkg/utils"
	"syscall"
	"time"
)

func main() {
	cfg := config.LoadConfig()
	utils.SetSecret(cfg.JWTSecret)

	// Initialize Logger
	logger.Init("development") // Change to "production" in prod env
	log := logger.Get()

	// Initialize Database with pgx/sqlc
	pgxPool, err := sqlcrepo.NewPgxPool(context.Background(), cfg.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	log.Info().Msg("Successfully connected to PostgreSQL via pgx/sqlc")

	// Initialize Repositories
	userRepo := sqlcrepo.NewUserRepository(pgxPool)
	productRepo := sqlcrepo.NewProductRepository(pgxPool)
	orderRepo := sqlcrepo.NewOrderRepository(pgxPool)
	txManager := sqlcrepo.NewTransactionManager(pgxPool)

	// Set up Router
	mux := http.NewServeMux()

	// --- Modules Initialization ---

	// Auth Module
	authUC := usecase.NewAuthUsecase(
		userRepo,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleTokenInfoURL,
		cfg.AccessTokenExpiry,
		cfg.RefreshTokenExpiry,
	)
	authHandler := v1.NewAuthHandler(authUC)

	// --- Storage Module (R2) ---
	r2Storage, err := storage.NewR2Storage(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2AccessKeySecret,
		cfg.R2BucketName,
		cfg.R2PublicURL,
	)
	if err != nil {
		log.Printf("Warning: Failed to initialize R2 Storage: %v", err)
	}
	uploadHandler := v1.NewUploadHandler(r2Storage)

	// Catalog Module
	catalogUC := usecase.NewCatalogUsecase(productRepo)
	catalogHandler := v1.NewCatalogHandler(catalogUC)

	// Admin Catalog Handlers
	adminCatalogHandler := v1.NewAdminCatalogHandler(catalogUC)

	// Order Module
	orderUC := usecase.NewOrderUsecase(orderRepo, productRepo, txManager)
	orderHandler := v1.NewOrderHandler(orderUC)
	adminOrderHandler := v1.NewAdminOrderHandler(orderUC)

	// Content Module
	contentRepo := sqlcrepo.NewContentRepository(pgxPool)
	contentUC := usecase.NewContentUsecase(contentRepo)
	contentHandler := v1.NewContentHandler(contentUC)

	// --- Routes ---

	// Auth
	mux.HandleFunc("POST /api/v1/auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/v1/auth/me", middleware.AuthMiddleware(http.HandlerFunc(authHandler.Me)))

	// User Profile / Address
	mux.Handle("POST /api/v1/user/addresses", middleware.AuthMiddleware(http.HandlerFunc(authHandler.AddAddress)))
	mux.Handle("GET /api/v1/user/addresses", middleware.AuthMiddleware(http.HandlerFunc(authHandler.GetAddresses)))

	// Uploads
	mux.Handle("POST /api/v1/upload", middleware.AuthMiddleware(http.HandlerFunc(uploadHandler.UploadFile)))

	// Content (Public)
	mux.HandleFunc("GET /api/v1/content/{key}", contentHandler.GetContent)

	// Catalog (Public)
	mux.HandleFunc("GET /api/v1/categories", catalogHandler.GetCategories)
	mux.HandleFunc("GET /api/v1/categories/tree", catalogHandler.GetCategories)
	mux.HandleFunc("GET /api/v1/products", catalogHandler.ListProducts)
	mux.HandleFunc("GET /api/v1/products/{slug}", catalogHandler.GetProductDetails)
	mux.HandleFunc("GET /api/v1/products/{id}/reviews", catalogHandler.GetReviews)                                          // Public
	mux.Handle("POST /api/v1/products/{id}/reviews", middleware.AuthMiddleware(http.HandlerFunc(catalogHandler.AddReview))) // Protected

	mux.HandleFunc("GET /api/v1/collections", catalogHandler.GetCollections)
	mux.HandleFunc("GET /api/v1/collections/{slug}", catalogHandler.GetCollectionBySlug)

	// Admin (Protected)
	adminMiddleware := func(h http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.AdminMiddleware(h))
	}

	// Admin Content
	mux.Handle("PUT /api/v1/admin/content/{key}", adminMiddleware(contentHandler.UpsertContent))

	// Admin Product Management
	mux.Handle("GET /api/v1/admin/products", adminMiddleware(adminCatalogHandler.ListProducts))
	mux.Handle("GET /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.GetProduct))
	mux.Handle("POST /api/v1/admin/products", adminMiddleware(adminCatalogHandler.CreateProduct))
	mux.Handle("PUT /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.UpdateProduct))
	mux.Handle("PATCH /api/v1/admin/products/{id}/status", adminMiddleware(adminCatalogHandler.UpdateProductStatus))
	mux.Handle("DELETE /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.DeleteProduct))
	mux.Handle("POST /api/v1/admin/inventory/adjust", adminMiddleware(adminCatalogHandler.AdjustStock))
	mux.Handle("GET /api/v1/admin/inventory/logs", adminMiddleware(adminCatalogHandler.GetInventoryLogs))

	mux.Handle("GET /api/v1/admin/categories", adminMiddleware(adminCatalogHandler.GetAllCategories))
	mux.Handle("POST /api/v1/admin/categories", adminMiddleware(adminCatalogHandler.CreateCategory))
	mux.Handle("PUT /api/v1/admin/categories/{id}", adminMiddleware(adminCatalogHandler.UpdateCategory))
	mux.Handle("DELETE /api/v1/admin/categories/{id}", adminMiddleware(adminCatalogHandler.DeleteCategory))
	mux.Handle("POST /api/v1/admin/categories/reorder", adminMiddleware(adminCatalogHandler.ReorderCategories))

	// Collections
	mux.Handle("GET /api/v1/admin/collections", adminMiddleware(adminCatalogHandler.GetAllCollections))
	mux.Handle("POST /api/v1/admin/collections", adminMiddleware(adminCatalogHandler.CreateCollection))
	mux.Handle("PUT /api/v1/admin/collections/{id}", adminMiddleware(adminCatalogHandler.UpdateCollection))
	mux.Handle("DELETE /api/v1/admin/collections/{id}", adminMiddleware(adminCatalogHandler.DeleteCollection))
	mux.Handle("POST /api/v1/admin/collections/{id}/products", adminMiddleware(adminCatalogHandler.ManageCollectionProduct))

	mux.Handle("GET /api/v1/admin/orders", adminMiddleware(adminOrderHandler.ListOrders))
	mux.Handle("PATCH /api/v1/admin/orders/{id}/status", adminMiddleware(adminOrderHandler.UpdateStatus))
	mux.Handle("GET /api/v1/admin/users", adminMiddleware(authHandler.ListUsers))

	// Cart & Order (Protected)
	mux.Handle("GET /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetCart)))

	mux.Handle("POST /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.AddToCart)))
	mux.Handle("DELETE /api/v1/cart/{productId}", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.RemoveFromCart)))
	mux.Handle("POST /api/v1/checkout", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.Checkout)))
	mux.Handle("GET /api/v1/orders", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetMyOrders)))

	// Health Check
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "db": "connected"}`))
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Info().Msgf("Server starting on %s", addr)

	// Apply CORS and Request Logger
	handler := middleware.CORS(mux)
	handler = middleware.RequestLogger(handler)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Graceful Shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	log.Info().Msgf("Server starting on %s", addr)

	// Wait for interrupt signal via channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}
