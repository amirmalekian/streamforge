.PHONY: help build test lint docker-up docker-down migrate-up migrate-down fmt vet

help:
	@echo "Available targets:"
	@echo "  build       - Build API and Worker binaries"
	@echo "  test        - Run unit tests with coverage"
	@echo "  lint        - Run golangci-lint"
	@echo "  fmt         - Format code with gofmt"
	@echo "  vet         - Run go vet"
	@echo "  docker-up   - Start docker-compose services"
	@echo "  docker-down - Stop docker-compose services"
	@echo "  migrate-up  - Run database migrations"
	@echo "  migrate-down - Rollback database migrations"

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o streamforge-api ./cmd/api
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o streamforge-worker ./cmd/worker

test:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

migrate-up:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path migrations -database "$(DATABASE_URL)" down

generate:
	go generate ./...

tidy:
	go mod tidy

clean:
	rm -f streamforge-api streamforge-worker coverage.out