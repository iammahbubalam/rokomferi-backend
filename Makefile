# ==========================================
# Rokomferi Backend Makefile
# ==========================================

# Load environment variables from .env
include .env
export

# Database URL (uses DB_DSN from .env)
DB_URL = $(DB_DSN)

# ==========================================
# Application
# ==========================================

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go

tidy:
	go mod tidy

test:
	go test ./... -v

# ==========================================
# Database Migrations
# ==========================================

migrateup:
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" up

migratedown:
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" down 1

migrateforce:
	@read -p "Enter version to force: " version; \
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" force $$version

migratestatus:
	$(HOME)/go/bin/migrate -path db/migrations -database "$(DB_URL)" version

migratecreate:
	@read -p "Enter migration name: " name; \
	$(HOME)/go/bin/migrate create -ext sql -dir db/migrations -seq $$name

# ==========================================
# Code Generation
# ==========================================

sqlc:
	$(HOME)/go/bin/sqlc generate

generate: sqlc
	@echo "Code generation complete."

# ==========================================
# Development Helpers
# ==========================================

dev: tidy sqlc run

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/

# ==========================================
# Docker (if using)
# ==========================================

docker-build:
	docker build -t rokomferi-backend .

docker-run:
	docker run -p 8080:8080 --env-file .env rokomferi-backend

# ==========================================
# Help
# ==========================================

help:
	@echo "Available commands:"
	@echo ""
	@echo "  Application:"
	@echo "    make run            - Run the application"
	@echo "    make build          - Build the binary"
	@echo "    make dev            - Tidy, generate, and run"
	@echo "    make tidy           - Run go mod tidy"
	@echo "    make test           - Run tests"
	@echo ""
	@echo "  Database:"
	@echo "    make migrateup      - Apply all pending migrations"
	@echo "    make migratedown    - Rollback last migration"
	@echo "    make migrateforce   - Force migration version"
	@echo "    make migratestatus  - Show current migration version"
	@echo "    make migratecreate  - Create a new migration"
	@echo ""
	@echo "  Code Generation:"
	@echo "    make sqlc           - Generate SQLC code"
	@echo "    make generate       - Run all code generators"
	@echo ""
	@echo "  Quality:"
	@echo "    make lint           - Run linter"
	@echo "    make fmt            - Format code"
	@echo ""
	@echo "  Docker:"
	@echo "    make docker-build   - Build Docker image"
	@echo "    make docker-run     - Run Docker container"

.PHONY: run build tidy test migrateup migratedown migrateforce migratestatus migratecreate sqlc generate dev lint fmt clean docker-build docker-run help
