# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=go-acs
BINARY_LINUX=$(BINARY_NAME)-linux
BINARY_WINDOWS=$(BINARY_NAME).exe

# Build the project
.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server

# Run the project
.PHONY: run
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server
	./$(BINARY_NAME)

# Development run with auto-reload (requires air)
.PHONY: dev
dev:
	air

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build files
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_LINUX)
	rm -f $(BINARY_WINDOWS)

# Test the project
.PHONY: test
test:
	$(GOTEST) -v ./...

# Test with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Build for Linux
.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) -v ./cmd/server

# Build for Windows
.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WINDOWS) -v ./cmd/server

# Build all platforms
.PHONY: build-all
build-all: build-linux build-windows

# Docker build
.PHONY: docker-build
docker-build:
	docker build -t go-acs .

# Docker run
.PHONY: docker-run
docker-run:
	docker-compose up -d

# Docker stop
.PHONY: docker-stop
docker-stop:
	docker-compose down

# Docker logs
.PHONY: docker-logs
docker-logs:
	docker-compose logs -f

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run

# Generate documentation
.PHONY: docs
docs:
	godoc -http=:6060

# Install air for development
.PHONY: install-air
install-air:
	$(GOCMD) install github.com/cosmtrek/air@latest

# Help
.PHONY: help
help:
	@echo "GO-ACS Makefile Commands:"
	@echo ""
	@echo "  make build        - Build the binary"
	@echo "  make run          - Build and run"
	@echo "  make dev          - Run with auto-reload (requires air)"
	@echo "  make deps         - Download dependencies"
	@echo "  make clean        - Clean build files"
	@echo "  make test         - Run tests"
	@echo "  make build-linux  - Build for Linux"
	@echo "  make build-windows- Build for Windows"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run with Docker Compose"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo ""
