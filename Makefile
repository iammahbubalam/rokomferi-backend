run:
	go run cmd/api/main.go

tidy:
	go mod tidy

build:
	go build -o bin/api cmd/api/main.go
