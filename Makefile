
# Load .env file if present
-include .env
export


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


clean: ## Remove build artifacts
	rm -rf bin/


lint: ## Run Go vet
	go vet ./...



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