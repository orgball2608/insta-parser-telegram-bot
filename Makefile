.PHONY: all build run test clean lint mock docker-build docker-run migrate-up migrate-down

# Go parameters
BINARY_NAME=insta-parser-bot
MAIN_PACKAGE=./cmd/main.go
COVERAGE_FILE=coverage.out

all: clean lint test build

build:
	@echo "Building..."
	go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

run:
	@echo "Running with air for hot reload..."
	air

test:
	@echo "Running tests..."
	go test -v -race -cover ./... -coverprofile=$(COVERAGE_FILE)
	go tool cover -func=$(COVERAGE_FILE)

clean:
	@echo "Cleaning..."
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)

lint:
	@echo "Running linters..."
	golangci-lint run ./...

mock:
	@echo "Generating mocks..."
	mockery

docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .

docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 $(BINARY_NAME)

migrate-up:
	@echo "Running database migrations..."
	go run tools/migrate/main.go up

migrate-down:
	@echo "Rolling back database migrations..."
	go run tools/migrate/main.go down

install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install github.com/vektra/mockery/v2@latest

help:
	@echo "Available commands:"
	@echo "  make build         - Build the application"
	@echo "  make run          - Run the application with hot reload"
	@echo "  make test         - Run tests with coverage"
	@echo "  make clean        - Clean build files"
	@echo "  make lint         - Run linters"
	@echo "  make mock         - Generate mocks"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo "  make install-tools- Install development tools"
