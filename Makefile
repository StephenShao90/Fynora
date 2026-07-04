SHELL := /bin/bash

.PHONY: install docker-up docker-down migrate seed api web test lint fmt build

install:
	cd apps/web && npm install
	cd services/api && go mod download

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down

migrate:
	docker compose exec -T postgres psql -U postgres -d fynora < services/api/migrations/001_init.sql

seed:
	@echo "Demo data is seeded automatically by POST /auth/demo-token."

api:
	cd services/api && go run ./cmd/api

web:
	cd apps/web && npm run dev

test:
	cd services/api && go test ./...

lint:
	cd services/api && go vet ./...
	cd apps/web && npm run lint

fmt:
	cd services/api && gofmt -w .

build:
	cd services/api && go build ./cmd/api
	cd apps/web && npm run build
