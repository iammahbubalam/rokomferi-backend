package main

import (
	"fmt"
	"log"
	"net/http"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/delivery/http/middleware"
	v1 "rokomferi-backend/internal/delivery/http/v1"
	"rokomferi-backend/internal/repository/postgres"
	"rokomferi-backend/internal/usecase"
	postgresPkg "rokomferi-backend/pkg/postgres"
)

func main() {
	cfg := config.LoadConfig()

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

	// 1. Auth Module
	userRepo := postgres.NewUserRepository(db)
	authUsecase := usecase.NewAuthUsecase(userRepo, cfg.GoogleClientID)
	authHandler := v1.NewAuthHandler(authUsecase)

	// 2. Catalog Module
	productRepo := postgres.NewProductRepository(db)
	catalogUC := usecase.NewCatalogUsecase(productRepo)
	catalogHandler := v1.NewCatalogHandler(catalogUC)

	// 3. Order Module
	orderRepo := postgres.NewOrderRepository(db)
	orderUC := usecase.NewOrderUsecase(orderRepo, productRepo)
	orderHandler := v1.NewOrderHandler(orderUC)

	// --- Routes ---

	// Auth
	mux.HandleFunc("POST /api/v1/auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/v1/auth/me", middleware.AuthMiddleware(http.HandlerFunc(authHandler.Me)))

	// Catalog
	mux.HandleFunc("GET /api/v1/categories/tree", catalogHandler.GetCategories)
	mux.HandleFunc("GET /api/v1/products", catalogHandler.ListProducts)
	mux.HandleFunc("GET /api/v1/products/{slug}", catalogHandler.GetProductDetails)

	// Cart & Order (Protected)
	mux.Handle("GET /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetCart)))
	mux.Handle("POST /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.AddToCart)))
	mux.Handle("POST /api/v1/checkout", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.Checkout)))

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

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
