.PHONY: run build test test-integration vet lint docker-build docker-up docker-down db-reset deploy-up deploy-down deploy-logs

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:
	go test -race -cover ./...

test-integration:
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:-postgres://tyrako:tyrako-dev-password@localhost:5432/tyrako?sslmode=disable}" TEST_REDIS_URL="$${TEST_REDIS_URL:-redis://localhost:6379/0}" go test -race -cover ./internal/repository/postgres/... ./internal/ratelimit/redislimit/...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

docker-build:
	docker build -f deployments/Dockerfile -t tyrako:latest .

docker-up:
	docker compose -f deployments/docker-compose.yml up -d --build

docker-down:
	docker compose -f deployments/docker-compose.yml down

deploy-up:
	docker compose --env-file .env.production -f deployments/docker-compose.prod.yml up -d --build

deploy-down:
	docker compose --env-file .env.production -f deployments/docker-compose.prod.yml down

deploy-logs:
	docker compose --env-file .env.production -f deployments/docker-compose.prod.yml logs -f --tail=100

db-reset:
	docker compose -f deployments/docker-compose.yml exec db psql -U tyrako -d tyrako -c "TRUNCATE api_keys RESTART IDENTITY;"
