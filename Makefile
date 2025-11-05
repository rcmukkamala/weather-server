.PHONY: help build run-server run-aggregator run-alarming run-notification run-dbwriter \
        docker-up docker-down docker-logs test clean kafka-topics kafka-init

# Default target
help:
	@echo "Weather Server - Available Commands"
	@echo "===================================="
	@echo "  make build              - Build all binaries"
	@echo "  make run-server         - Run TCP server"
	@echo "  make run-dbwriter       - Run database writer service"
	@echo "  make run-aggregator     - Run aggregation service"
	@echo "  make run-alarming       - Run alarming service"
	@echo "  make run-notification   - Run notification service"
	@echo "  make docker-up          - Start all Docker services"
	@echo "  make docker-down        - Stop all Docker services"
	@echo "  make docker-logs        - View Docker logs"
	@echo "  make kafka-topics       - List Kafka topics"
	@echo "  make kafka-init         - Manually initialize Kafka topics"
	@echo "  make test               - Run tests"
	@echo "  make clean              - Clean build artifacts"

# Build all binaries
build:
	@echo "Building binaries..."
	go build -o bin/server ./cmd/server
	go build -o bin/dbwriter ./cmd/dbwriter
	go build -o bin/aggregator ./cmd/aggregator
	go build -o bin/alarming ./cmd/alarming
	go build -o bin/notification ./cmd/notification
	@echo "Build complete!"

# Run services
run-server: build
	./bin/server

run-dbwriter: build
	./bin/dbwriter

run-aggregator: build
	./bin/aggregator

run-alarming: build
	./bin/alarming

run-notification: build
	./bin/notification

# Docker commands
docker-up:
	docker-compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@echo "Docker services started!"

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-restart:
	docker-compose restart

# Testing
test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Initialize (run migrations)
init: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Database is ready. Run 'make run-server' to start the server."

# Quick start (all services)
start-all: docker-up
	@echo "Starting all services in background..."
	@sleep 5
	@nohup ./bin/server > logs/server.log 2>&1 &
	@nohup ./bin/aggregator > logs/aggregator.log 2>&1 &
	@nohup ./bin/alarming > logs/alarming.log 2>&1 &
	@nohup ./bin/notification > logs/notification.log 2>&1 &
	@echo "All services started! Check logs/ directory for output."

# Development dependencies
deps:
	go mod download
	go mod tidy

# Kafka topic management
kafka-topics:
	@echo "Listing Kafka topics..."
	@docker exec weather-kafka /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 --list

kafka-init:
	@echo "Initializing Kafka topics..."
	@docker exec weather-kafka /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create --if-not-exists \
		--topic weather.metrics.raw \
		--partitions 10 \
		--replication-factor 1
	@docker exec weather-kafka /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create --if-not-exists \
		--topic weather.alarms \
		--partitions 10 \
		--replication-factor 1
	@echo "✓ Kafka topics initialized!"
	@make kafka-topics

