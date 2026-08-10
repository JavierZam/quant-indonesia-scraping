.PHONY: run build test lint clean docker-build docker-run migrate-up migrate-down help

# Variables
APP_NAME := quant-intel-server
DOCKER_IMAGE := quant-indonesia-scraping
DB_DSN := postgres://quantuser:quantpass@localhost:5432/quantintel?sslmode=disable
MIGRATE_CMD := migrate -database "$(DB_DSN)" -path migrations

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /' 

## run: Run the server locally
run:
	go run ./cmd/server

## build: Build the server binary
build:
	CGO_ENABLED=0 go build -ldflags='-s -w' -o $(APP_NAME) ./cmd/server

## test: Run all tests
test:
	go test ./... -v -race -count=1

## test-coverage: Run tests with coverage report
test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	rm -f $(APP_NAME) coverage.out coverage.html

## docker-build: Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

## docker-run: Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env --network quant-indonesia-scraping_quantnet $(DOCKER_IMAGE)

## infra-up: Start infrastructure (Postgres + Valkey)
infra-up:
	docker compose up -d

## infra-down: Stop infrastructure
infra-down:
	docker compose down

## migrate-up: Run all database migrations
migrate-up:
	@echo "Running migrations..."
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000001_create_stocks_table.up.sql
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000002_create_executives_table.up.sql
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000003_create_news_articles_table.up.sql
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000004_create_news_stock_tags_table.up.sql
	@echo "Migrations complete!"

## migrate-down: Rollback all database migrations
migrate-down:
	@echo "Rolling back migrations..."
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000004_create_news_stock_tags_table.down.sql
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000003_create_news_articles_table.down.sql
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000002_create_executives_table.down.sql
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f migrations/000001_create_stocks_table.down.sql
	@echo "Rollback complete!"

## seed: Seed initial Indonesian stock data
seed:
	@echo "Seeding stock data..."
	@PGPASSWORD=quantpass psql -h localhost -U quantuser -d quantintel -f scripts/seed_stocks.sql
	@echo "Seed complete!"
