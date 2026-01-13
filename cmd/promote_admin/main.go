package main

import (
	"log"
	"os"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/domain"
	postgresPkg "rokomferi-backend/pkg/postgres"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <email>")
	}
	email := os.Args[1]

	cfg := config.LoadConfig()
	db, err := postgresPkg.NewClient(cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	var user domain.User
	result := db.Where("email = ?", email).First(&user)

	if result.Error != nil {
		log.Printf("User %s not found. Creating...", email)
		// Create new admin user
		user = domain.User{
			ID:        "u_" + email, // Simple ID for now or UUID
			Email:     email,
			Role:      "admin",
			FirstName: "Admin",
			LastName:  "User",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create admin user: %v", err)
		}
		log.Printf("User %s created as ADMIN.", email)
	} else {
		// Update existing
		user.Role = "admin"
		if err := db.Save(&user).Error; err != nil {
			log.Fatalf("Failed to update user role: %v", err)
		}
		log.Printf("User %s promoted to ADMIN.", email)
	}
}
