# CirrusSync API Makefile
# Common development tasks for the CirrusSync API project

.PHONY: help build run dev test test-cover clean docker-build docker-run docker-stop \
        deps fmt lint vet keys setup db-migrate db-reset logs air install-tools \
        check security docker-clean prod-build

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=cirrussync-api
MAIN_PATH=cmd/main.go
BUILD_DIR=build
DOCKER_IMAGE=cirrussync-api
DOCKER_TAG=latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-w -s"
BUILD_FLAGS=-a -installsuffix cgo

## help: Show this help message
help:
	@echo "CirrusSync API Development Commands"
	@echo "=================================="
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## setup: Run the setup script to initialize the development environment
setup:
	@./scripts/setup.sh

## build: Build the application binary
build: clean
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## prod-build: Build optimized binary for production
prod-build: clean
	@echo "Building production binary..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags "-w -s -extldflags '-static'" \
		-a -installsuffix cgo \
		-o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Production build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run the application with hot reload using Air
dev:
	@echo "Starting development server with hot reload..."
	@air

## air: Alias for dev command
air: dev

## test: Run all tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

## test-cover: Run tests with coverage report
test-cover:
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -cover -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-race: Run tests with race condition detection
test-race:
	@echo "Running tests with race detection..."
	@$(GOTEST) -v -race ./...

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify
	@$(GOMOD) tidy

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@$(GOCMD) fmt ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@golangci-lint run

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@$(GOVET) ./...

## check: Run all code quality checks
check: fmt vet lint test

## security: Run security checks
security:
	@echo "Running security checks..."
	@gosec ./...

## clean: Clean build artifacts and temporary files
clean:
	@echo "Cleaning up..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf tmp/
	@rm -f coverage.out coverage.html
	@rm -f *.log

## keys: Generate RSA keys for JWT
keys:
	@echo "Generating RSA keys..."
	@mkdir -p keys
	@openssl genrsa -out keys/private.pem 2048
	@openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@chmod 600 keys/private.pem
	@chmod 644 keys/public.pem
	@echo "RSA keys generated successfully"

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@$(GOCMD) install github.com/cosmtrek/air@latest
	@$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GOCMD) install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "Development tools installed"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

## docker-run: Run the application in Docker
docker-run:
	@echo "Starting application with Docker Compose..."
	@docker-compose up -d

## docker-dev: Run development environment with Docker
docker-dev:
	@echo "Starting development environment..."
	@docker-compose up postgres redis minio mailhog -d
	@echo "Development services started. You can now run 'make dev' or 'make run'"

## docker-stop: Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

## docker-clean: Clean Docker resources
docker-clean:
	@echo "Cleaning Docker resources..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f

## docker-logs: Show Docker logs
docker-logs:
	@docker-compose logs -f cirrussync-api

## db-migrate: Run database migrations
db-migrate:
	@echo "Running database migrations..."
	@docker-compose exec cirrussync-api ./cirrussync-api migrate up

## db-reset: Reset database (WARNING: This will delete all data)
db-reset:
	@echo "WARNING: This will delete all database data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo ""; \
		docker-compose down -v; \
		docker-compose up postgres -d; \
		sleep 5; \
		docker-compose up cirrussync-api -d; \
	else \
		echo ""; \
		echo "Database reset cancelled."; \
	fi

## logs: Show application logs
logs:
	@docker-compose logs -f cirrussync-api

## ps: Show running containers
ps:
	@docker-compose ps

## shell: Open shell in the API container
shell:
	@docker-compose exec cirrussync-api /bin/sh

## db-shell: Connect to PostgreSQL database
db-shell:
	@docker-compose exec postgres psql -U cirrussync -d cirrussync

## redis-cli: Connect to Redis CLI
redis-cli:
	@docker-compose exec redis redis-cli

## env: Create .env file from template
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created from template. Please edit it with your values."; \
	else \
		echo ".env file already exists."; \
	fi

## health: Check service health
health:
	@echo "Checking service health..."
	@curl -f http://localhost:8000/health || echo "API is not responding"
	@docker-compose ps

## size: Show binary size
size: build
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

## mod-graph: Show module dependency graph
mod-graph:
	@$(GOCMD) mod graph

## mod-why: Show why a module is needed
mod-why:
	@read -p "Enter module name: " module; \
	$(GOCMD) mod why $$module

## profile: Run with profiling enabled
profile:
	@echo "Starting with profiling enabled..."
	@$(GOCMD) run -ldflags "-X main.enableProfiling=true" $(MAIN_PATH)

## update-deps: Update all dependencies
update-deps:
	@echo "Updating dependencies..."
	@$(GOCMD) get -u ./...
	@$(GOMOD) tidy

## vendor: Create vendor directory
vendor:
	@echo "Creating vendor directory..."
	@$(GOMOD) vendor

## docs: Generate documentation
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060
	@echo "Documentation server started at http://localhost:6060"

## todo: Show TODO comments in code
todo:
	@echo "TODO items in code:"
	@grep -r "TODO\|FIXME\|HACK" --include="*.go" . || echo "No TODO items found"

## lines: Count lines of code
lines:
	@echo "Lines of code:"
	@find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
