package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	DBUrl              string
	GoogleClientID     string
	JWTSecret          string
	AllowedOrigin      string
	GoogleTokenInfoURL string
	GoogleClientSecret string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	// DB Config
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnIdleTime time.Duration
	// R2 Storage
	R2AccountID       string
	R2AccessKeyID     string
	R2AccessKeySecret string
	R2BucketName      string
	R2PublicURL       string
}

func LoadConfig() *Config {
	// Load .env file if it exists (mainly for local dev)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, relying on system env vars")
	}

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DBUrl:              getEnv("DB_DSN", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		JWTSecret:          getEnv("JWT_SECRET", "default_secret_CHANGE_ME"),
		AllowedOrigin:      getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		GoogleTokenInfoURL: getEnv("GOOGLE_TOKEN_INFO_URL", "https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s"),
		AccessTokenExpiry:  getDurationEnv("ACCESS_TOKEN_EXPIRY", time.Hour*24),    // Default 24h
		RefreshTokenExpiry: getDurationEnv("REFRESH_TOKEN_EXPIRY", time.Hour*24*7), // Default 7d

		DBMaxConns:        getInt32Env("DB_MAX_CONNS", 50),
		DBMinConns:        getInt32Env("DB_MIN_CONNS", 10),
		DBMaxConnIdleTime: getDurationEnv("DB_MAX_CONN_IDLE_TIME", time.Minute*15),

		// R2 Storage
		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2AccessKeySecret: getEnv("R2_ACCESS_KEY_SECRET", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),
	}

	cfg.Validate()
	return cfg
}

func (c *Config) Validate() {
	if c.DBUrl == "" {
		log.Fatal("CRITICAL: DB_DSN environment variable is required")
	}
	if c.JWTSecret == "default_secret_CHANGE_ME" {
		log.Println("WARNING: Using default JWT secret. Setting up for failure in production.")
	}
	if c.GoogleClientID == "" {
		log.Fatal("CRITICAL: GOOGLE_CLIENT_ID is required")
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
		log.Printf("Invalid duration for %s, using fallback", key)
	}
	return fallback
}
