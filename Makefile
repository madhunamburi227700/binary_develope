# AI Guardian API - Development Makefile
.PHONY: help build up down logs clean restart status test fmt deps migrate db redis

# Docker Commands
build: ## Build all Docker images
	@echo "Building Docker images..."
	cd build && docker build -t ai-guardian-api -f Dockerfile ..

up: ## Start all Docker services
	@echo "Starting Docker services..."
	cd build && docker-compose up -d

down: ## Stop all Docker services
	@echo "Stopping Docker services..."
	cd build && docker-compose down

logs: ## View logs
	cd build && docker-compose logs -f