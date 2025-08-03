# Makefile for Rate Limiter System

.PHONY: help build run test clean docker setup-deps

# Default target
help:
	@echo "Available commands:"
	@echo "  setup-deps     - Initialize Go modules and dependencies"
	@echo "  build          - Build all services"
	@echo "  run-basic      - Run basic rate limiter service"
	@echo "  run-integrated - Run integrated service with rule engine"
	@echo "  test           - Run all tests"
	@echo "  test-rate      - Run rate limiter tests"
	@echo "  test-rules     - Run rule engine tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-run     - Run services in Docker"
	@echo "  stack-up       - Start full stack with cleanup"
	@echo "  stack-down     - Stop full stack"
	@echo "  network-clean  - Clean up Docker networks"
	@echo "  network-check  - Check for network conflicts"

# Setup dependencies
setup-deps:
	@echo "Setting up Go modules..."
	cd rate-limiter && go mod tidy
	cd rule-engine && go mod tidy
	@echo "Dependencies setup complete"

# Build all services
build: setup-deps
	@echo "Building rate limiter service..."
	cd rate-limiter && go build -o bin/rate-limiter-server cmd/server/main.go
	@echo "Building rule engine..."
	cd rule-engine && go build -o bin/rule-engine ./...
	@echo "Building integrated service..."
	cd rate-limiter && go build -o bin/integrated-server cmd/integrated-server/main.go
	@echo "Build complete"

# Run basic rate limiter service
run-basic: setup-deps
	@echo "Starting basic rate limiter service on :8080..."
	cd rate-limiter && go run cmd/server/main.go

# Run integrated service
run-integrated: setup-deps
	@echo "Starting integrated service on :8080..."
	cd rate-limiter && go run cmd/integrated-server/*.go

# Run all tests
test: test-rate test-rules

# Run rate limiter tests
test-rate:
	@echo "Running rate limiter tests..."
	cd rate-limiter && go test -v ./...

# Run rule engine tests
test-rules:
	@echo "Running rule engine tests..."
	cd rule-engine && go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf rate-limiter/bin/
	rm -rf rule-engine/bin/
	rm -rf bin/
	cd rate-limiter && go clean
	cd rule-engine && go clean
	@echo "Clean complete"

# Docker targets
docker-build:
	@echo "Building Docker images..."
	docker build -t rate-limiter:latest -f docker/Dockerfile.rate-limiter .
	docker build -t integrated-service:latest -f docker/Dockerfile.integrated .
	docker build -t test-client:latest -f docker/Dockerfile.test-client .

docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker services..."
	docker-compose down

docker-restart: docker-stop docker-run

docker-logs:
	@echo "Showing Docker logs..."
	docker-compose logs -f

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f

# Full stack operations
stack-up: docker-build network-clean
	@echo "Starting full stack..."
	chmod +x docker/network-cleanup.sh
	./docker/network-cleanup.sh
	docker-compose up -d
	@echo ""
	@echo "🚀 Full stack is starting up!"
	@echo ""
	@echo "Services will be available at:"
	@echo "  Rate Limiter API:    http://localhost:8080"
	@echo "  Integrated API:      http://localhost:8081"
	@echo "  Load Balancer:       http://localhost:80"
	@echo "  Grafana:            http://localhost:3000 (admin/admin123)"
	@echo "  Prometheus:         http://localhost:9090"
	@echo "  Kibana:             http://localhost:5601"
	@echo "  Jaeger:             http://localhost:16686"
	@echo "  Admin UI:           http://localhost:3001"
	@echo ""
	@echo "Databases:"
	@echo "  PostgreSQL:         localhost:5432 (ratelimiter/ratelimiter123)"
	@echo "  Redis:              localhost:6379"
	@echo "  Elasticsearch:      localhost:9200"
	@echo ""

stack-down:
	@echo "Stopping full stack..."
	docker-compose down
	@echo "Full stack stopped"

stack-restart: stack-down stack-up

stack-status:
	@echo "Stack status:"
	docker-compose ps

# Testing with Docker
test-load:
	@echo "Running load tests..."
	docker-compose --profile testing up test-client

test-integration:
	@echo "Running integration tests with Docker..."
	@echo "Cleaning up any existing networks..."
	-docker network prune -f
	docker-compose up -d postgres redis
	sleep 10
	docker-compose run --rm rate-limiter-service go test ./...
	docker-compose run --rm integrated-service go test ./...
	docker-compose down

# Network management
network-clean:
	@echo "Cleaning up Docker networks..."
	chmod +x docker/network-cleanup.sh
	./docker/network-cleanup.sh

network-check:
	@echo "Checking network conflicts..."
	docker network ls
	@echo ""
	@echo "Checking IP subnet usage..."
	docker network inspect $$(docker network ls -q) 2>/dev/null | grep -E '"Subnet"|"Name"' | paste - - | head -10 || true

# Monitoring
monitor-logs:
	@echo "Monitoring logs..."
	docker-compose logs -f rate-limiter-service integrated-service

monitor-metrics:
	@echo "Opening Grafana for metrics monitoring..."
	open http://localhost:3000

monitor-traces:
	@echo "Opening Jaeger for trace monitoring..."
	open http://localhost:16686

# Development helpers
dev-setup: setup-deps
	@echo "Setting up development environment..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/lint/golint@latest
	@echo "Development setup complete"

# Format code
fmt:
	@echo "Formatting code..."
	cd rate-limiter && go fmt ./...
	cd rule-engine && go fmt ./...
	gofmt -w cmd/

# Lint code
lint:
	@echo "Linting code..."
	cd rate-limiter && golint ./...
	cd rule-engine && golint ./...

# Generate documentation
docs:
	@echo "Generating documentation..."
	cd rate-limiter && go doc -all ./... > docs/rate-limiter.md
	cd rule-engine && go doc -all ./... > docs/rule-engine.md

# Performance testing
perf-test:
	@echo "Running performance tests..."
	cd rate-limiter && go test -bench=. -benchmem ./...

# Create release
release: clean build test
	@echo "Creating release..."
	mkdir -p release
	cp bin/* release/
	tar -czf release/rate-limiter-$(shell date +%Y%m%d).tar.gz release/
	@echo "Release created in release/ directory"

# Quick start for new developers
quick-start: setup-deps build
	@echo ""
	@echo "🚀 Quick Start Complete!"
	@echo ""
	@echo "Try these commands:"
	@echo "  make run-basic      # Start basic rate limiter"
	@echo "  make run-integrated # Start integrated service"
	@echo ""
	@echo "API will be available at http://localhost:8080"
	@echo "Swagger UI will be available at http://localhost:8081/swagger/"
	@echo ""
