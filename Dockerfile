# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o go-acs cmd/server/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Copy binary from builder
COPY --from=builder /app/go-acs .

# Copy web assets
COPY --from=builder /app/web ./web

# Create data directory
RUN mkdir -p /app/data

# Expose ports
EXPOSE 8080 7547

# Environment variables
ENV SERVER_PORT=8080
ENV TR069_PORT=7547
ENV DATABASE_URL=/app/data/goacs.db
ENV LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./go-acs"]
