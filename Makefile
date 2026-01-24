.PHONY: build run test clean schema-docs docker-up docker-down

# Build the application
build:
	go build -o bin/kitamanager-api ./cmd/api

# Run the application locally
run:
	go run ./cmd/api

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Update database schema documentation (requires running database)
schema-docs:
	tbls doc --force

# Start docker containers
docker-up:
	docker compose up -d

# Stop docker containers
docker-down:
	docker compose down

# Rebuild and restart docker containers
docker-rebuild:
	docker compose up -d --build
