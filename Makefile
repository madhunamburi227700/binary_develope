# AI Guardian API - Development Makefile
.PHONY: help build up down logs clean restart status test fmt deps migrate db redis swagger

# Docker Commands
build: ## Build all Docker images
	@echo "Building Docker images..."
	docker build --platform linux/amd64 --secret id=GIT_TOKEN,src=token -t ai-guardian-api -f build/Dockerfile .

up: ## Start all Docker services
	@echo "Starting Docker services..."
	cd build && docker-compose up -d

down: ## Stop all Docker services
	@echo "Stopping Docker services..."
	cd build && docker-compose down

logs: ## View logs
	cd build && docker-compose logs -f

test:
	go clean -testcache && go test ./... -cover

swagger: ## Regenerate swagger docs (docs/swagger.json, docs/swagger.yaml)
	swag init -g cmd/main.go -o docs --parseDependency --parseInternal