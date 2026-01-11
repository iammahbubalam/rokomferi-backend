package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN not found")
	}

	// Connect to the configured DB (rokomferi) so we can drop the other one
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Attempting to drop database 'neondb'...")

	// 1. Terminate connections to neondb to avoid "database is being accessed by other users" error
	// This query kills all connections to neondb
	err = db.Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'neondb'").Error
	if err != nil {
		log.Printf("Warning: Failed to terminate connections (maybe none exist): %v", err)
	}

	// 2. Drop the database
	// DROP DATABASE cannot run inside a transaction, so ensuring we use raw exec outside of one.
	// GORM's Exec usually works, but to be safe we can use the underlying sql DB if needed.
	// But db.Exec() is standard.
	if err := db.Exec("DROP DATABASE IF EXISTS neondb").Error; err != nil {
		log.Fatal("Failed to drop database neondb: ", err)
	}

	log.Println("✅ Database 'neondb' successfully deleted.")
}
