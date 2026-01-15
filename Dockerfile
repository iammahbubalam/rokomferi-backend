# Stage 1: Build
FROM golang:alpine AS builder

# Install git + SSL ca certificates.
# Git is required for fetching Go dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
# CGO_ENABLED=0: Disable CGO for a statically linked binary (no external dependencies)
# -ldflags="-w -s": Strip debug information to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api/main.go

# Stage 2: Production Runtime
FROM alpine:latest

WORKDIR /root/

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

CMD ["./main"]
