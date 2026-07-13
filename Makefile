SHELL := /bin/bash

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: install docker-up docker-down migrate seed api worker web local-env smoke test lint fmt build

install:
	cd apps/web && npm install
	cd services/api && go mod download

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

migrate:
	docker compose exec -T postgres psql -U postgres -d fynora < services/api/migrations/001_init.sql
	docker compose exec -T postgres psql -U postgres -d fynora < services/api/migrations/002_phase3_reliability.sql
	docker compose exec -T postgres psql -U postgres -d fynora < services/api/migrations/003_phase4_operability.sql
	docker compose exec -T postgres psql -U postgres -d fynora < services/api/migrations/004_phase7_integrations.sql

seed:
	@echo "Demo data is seeded automatically by POST /auth/demo-token."

api:
	cd services/api && go run ./cmd/api

worker:
	cd services/api && go run ./cmd/worker

web:
	cd apps/web && npm run dev

local-env:
	node scripts/use-local-env.mjs

smoke:
	node scripts/smoke-clearflow.mjs

test:
	cd services/api && go test ./...

lint:
	cd services/api && go vet ./...
	cd apps/web && npm run lint

fmt:
	cd services/api && gofmt -w .

build:
	cd services/api && go build ./cmd/api
	cd services/api && go build ./cmd/worker
	cd apps/web && npm run build
