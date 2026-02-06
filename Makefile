.PHONY: build run test clean docker-build docker-up docker-down

# Build the application
build:
	go build -o bin/comment ./cmd/main.go

# Run the application
run:
	go run ./cmd/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Docker commands
docker-build:
	docker build -t minisource/comment:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-dev:
	docker-compose -f docker-compose.dev.yml up -d

docker-logs:
	docker-compose logs -f comment

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/main.go -o docs --parseDependency --parseInternal

# Generate mocks (if needed)
mocks:
	mockgen -source=internal/usecase/comment_usecase.go -destination=internal/mocks/mock_notifier.go -package=mocks
