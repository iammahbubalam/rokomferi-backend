package main

import (
	"fmt"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var dbs []string
	// Allow connection to list all dbs (pg_database)
	if err := db.Raw("SELECT datname FROM pg_database WHERE datistemplate = false").Scan(&dbs).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Println("Available Databases:")
	for _, name := range dbs {
		fmt.Printf("- %s\n", name)
	}
}
