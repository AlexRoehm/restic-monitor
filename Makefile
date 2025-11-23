.PHONY: help build run dev clean test docker docker-build docker-push frontend backend install deps

# Variables
BINARY_NAME=restic-monitor
DOCKER_IMAGE=guxxde/restic-monitor
DOCKER_TAG=latest
GO_FILES=$(shell find . -name '*.go' -type f)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install Go dependencies
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy

install: deps ## Install project dependencies (Go + npm)
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

build: backend frontend ## Build both backend and frontend
	@echo "Build complete!"

backend: ## Build Go backend binary
	@echo "Building backend..."
	go build -o $(BINARY_NAME) cmd/restic-monitor/main.go

frontend: ## Build frontend for production
	@echo "Building frontend..."
	cd frontend && npm run build

run: ## Run the application with .env configuration
	@echo "Running application..."
	export $$(cat .env | grep -v '^#' | xargs) && go run cmd/restic-monitor/main.go

dev: ## Run backend and frontend in development mode (requires two terminals)
	@echo "Start backend: make dev-backend"
	@echo "Start frontend: make dev-frontend"

dev-backend: ## Run backend in development mode
	export $$(cat .env | grep -v '^#' | xargs) && go run cmd/restic-monitor/main.go

dev-frontend: ## Run frontend development server
	cd frontend && npm run dev

watch: ## Watch and rebuild backend on file changes
	@echo "Watching for changes..."
	@command -v air >/dev/null 2>&1 || { echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; exit 1; }
	export $$(cat .env | grep -v '^#' | xargs) && air

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf frontend/dist
	rm -rf data/*.db
	rm -rf public/*.txt

test: ## Run Go tests
	@echo "Running tests..."
	go test -v ./...

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-push: docker-build ## Build and push Docker image
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-run: ## Run application in Docker
	@echo "Running in Docker..."
	docker-compose up

docker-stop: ## Stop Docker containers
	docker-compose down

docker: docker-build docker-run ## Build and run in Docker

lint: ## Run Go linter
	@echo "Running linter..."
	golangci-lint run

format: ## Format Go code
	@echo "Formatting code..."
	gofmt -s -w .
	cd frontend && npm run format || true

swagger: ## Generate Swagger/OpenAPI specification from code annotations
	@echo "Generating Swagger specification..."
	@command -v swag >/dev/null 2>&1 || { echo "swag not found. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init -g cmd/restic-monitor/main.go -o api --parseDependency --parseInternal

.DEFAULT_GOAL := help
