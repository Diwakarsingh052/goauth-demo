


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

docker-down: ## Stop and remove all containers
	docker compose down


docker-restart: ## Restart all services
	docker compose restart

docker-clean: ## Stop containers, remove volumes, and clean images
	docker compose down -v --rmi local
	@echo "All containers, volumes, and images removed."