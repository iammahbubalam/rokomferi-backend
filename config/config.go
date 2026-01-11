package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Env            string
	DBUrl          string
	GoogleClientID string
	JWTSecret      string
}

func LoadConfig() *Config {
	// Load .env file if it exists (mainly for local dev)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, relying on system env vars")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		Env:            getEnv("ENV", "development"),
		DBUrl:          getEnv("DB_DSN", ""),
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
		JWTSecret:      getEnv("JWT_SECRET", "default_secret_CHANGE_ME"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
