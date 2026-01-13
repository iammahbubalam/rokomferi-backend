package main

import (
	"log"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/domain"
	postgresPkg "rokomferi-backend/pkg/postgres"
)

func main() {
	log.Println("--- Starting Database Migration ---")
	cfg := config.LoadConfig()
	db, err := postgresPkg.NewClient(cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Migrating Users...")
	if err := db.AutoMigrate(&domain.User{}, &domain.Address{}, &domain.RefreshToken{}); err != nil {
		log.Printf("Error migrating users: %v", err)
	}

	log.Println("Migrating Catalog (Categories, Products, Collections)...")
	// Note: AutoMigrate will create join tables for Many2Many automatically
	if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Variant{}, &domain.InventoryLog{}, &domain.Review{}, &domain.Collection{}); err != nil {
		log.Printf("Error migrating catalog: %v", err)
	}

	log.Println("Migrating Orders...")
	if err := db.AutoMigrate(&domain.Cart{}, &domain.CartItem{}, &domain.Order{}, &domain.OrderItem{}); err != nil {
		log.Printf("Error migrating orders: %v", err)
	}

	log.Println("--- Migration Completed Successfully ---")
}
