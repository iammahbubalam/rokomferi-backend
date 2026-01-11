package main

import (
	"flag"
	"log"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/pkg/postgres"
)

func main() {
	email := flag.String("email", "", "Email of the user to promote to admin")
	flag.Parse()

	if *email == "" {
		log.Fatal("Please provide an email using -email flag")
	}

	cfg := config.LoadConfig()

	// Connect to DB
	db, err := postgres.NewClient(cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Update User
	result := db.Model(&domain.User{}).Where("email = ?", *email).Update("role", "admin")

	if result.Error != nil {
		log.Fatalf("Failed to update user: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		log.Printf("No user found with email: %s", *email)
	} else {
		log.Printf("Successfully promoted %s to admin!", *email)
	}
}
