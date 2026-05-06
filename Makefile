.PHONY: build run-api run-web test test-unit test-integration clean setup migrate \
       deps lint help docker-build docker-up docker-down docker-logs docker-clean

# Load .env file if present
-include .env
export

# ─────────────────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────────────────
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Local Development:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -v docker | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Docker:"
	@grep -E '^docker-[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─────────────────────────────────────────────────────────────────────────────
# Local Development
# ─────────────────────────────────────────────────────────────────────────────
build: ## Build both API and Web server binaries
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/web ./cmd/web
	@echo "Binaries built in ./bin/"

run-api: ## Run the REST API server (default port 8080)
	go run ./cmd/api

run-web: ## Run the Web frontend server (default port 8081)
	go run ./cmd/web

test: ## Run all tests (unit + integration if DB available)
	@echo "Running unit tests..."
	go test -v ./internal/api/middleware/...
	@echo ""
	@echo "Note: Handler tests require a MySQL test database."
	@echo "Set TEST_DB_DSN env var or ensure challange_go_test database exists."
	@echo "Run: make test-integration"

test-unit: ## Run unit tests only (no database required)
	go test -v ./internal/api/middleware/...

test-integration: ## Run integration tests (requires MySQL)
	go test -v ./internal/api/handler/...

clean: ## Remove build artifacts
	rm -rf bin/

setup: ## Create MySQL databases for development and testing
	@echo "Creating databases..."
	mysql -u $${DB_USER:-root} -e "CREATE DATABASE IF NOT EXISTS $${DB_NAME:-challange_go};" 2>/dev/null || \
		echo "Could not create dev database. Check MySQL connection."
	mysql -u $${DB_USER:-root} -e "CREATE DATABASE IF NOT EXISTS challange_go_test;" 2>/dev/null || \
		echo "Could not create test database. Check MySQL connection."
	@echo "Done. Copy .env.example to .env and configure your settings."

migrate: ## Run database migration (also runs automatically on API start)
	@echo "Migrations run automatically when the API server starts."
	@echo "To manually create the schema, run:"
	@echo "  mysql -u root challange_go < migrations/001_init.sql"

deps: ## Download Go dependencies
	go mod tidy
	go mod download

lint: ## Run Go vet
	go vet ./...

# ─────────────────────────────────────────────────────────────────────────────
# Docker
# ─────────────────────────────────────────────────────────────────────────────
docker-build: ## Build Docker images for API and Web
	docker compose build

docker-up: ## Start all services (MySQL + API + Web) in containers
	docker compose up --build -d
	@echo ""
	@echo "Services starting..."
	@echo "  MySQL:  localhost:$${DB_PORT:-3306}"
	@echo "  API:    http://localhost:$${API_PORT:-8080}"
	@echo "  Web:    http://localhost:$${WEB_PORT:-8081}"
	@echo ""
	@echo "Run 'make docker-logs' to view logs."

docker-down: ## Stop and remove all containers
	docker compose down

docker-logs: ## Tail logs from all containers
	docker compose logs -f

docker-restart: ## Restart all services
	docker compose restart

docker-clean: ## Stop containers, remove volumes, and clean images
	docker compose down -v --rmi local
	@echo "All containers, volumes, and images removed."