SHELL := /bin/bash

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: install docker-up docker-down ensure-db migrate seed api worker web dev dev-smoke local-env smoke test lint fmt build verify verify-backend verify-frontend verify-scripts compose-check vuln

install:
	cd apps/web && npm install
	cd services/api && go mod download

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

ensure-db:
	@docker compose exec -T postgres psql -U postgres -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'clearflow'" | grep -q 1 || docker compose exec -T postgres createdb -U postgres clearflow

migrate: ensure-db
	docker compose exec -T postgres psql -U postgres -d clearflow < services/api/migrations/001_init.sql
	docker compose exec -T postgres psql -U postgres -d clearflow < services/api/migrations/002_phase3_reliability.sql
	docker compose exec -T postgres psql -U postgres -d clearflow < services/api/migrations/003_phase4_operability.sql
	docker compose exec -T postgres psql -U postgres -d clearflow < services/api/migrations/004_phase7_integrations.sql
	docker compose exec -T postgres psql -U postgres -d clearflow < services/api/migrations/005_phase8_production_readiness.sql
	docker compose exec -T postgres psql -U postgres -d clearflow < services/api/migrations/006_product_readiness.sql

seed:
	@echo "Demo data is seeded automatically by POST /auth/demo-token."

api:
	cd services/api && go run ./cmd/api

worker:
	cd services/api && go run ./cmd/worker

web:
	cd apps/web && npm run dev

dev:
	node scripts/dev.mjs

dev-smoke:
	node scripts/dev.mjs --smoke

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
	cd services/api && go build -o /tmp/clearflow-api ./cmd/api
	cd services/api && go build -o /tmp/clearflow-worker ./cmd/worker
	cd apps/web && npm run build

verify: verify-backend verify-frontend verify-scripts compose-check

verify-backend:
	cd services/api && gofmt -w .
	cd services/api && go test ./...
	cd services/api && go vet ./...
	cd services/api && go build -o /tmp/clearflow-api ./cmd/api
	cd services/api && go build -o /tmp/clearflow-worker ./cmd/worker

verify-frontend:
	cd apps/web && npm run lint
	cd apps/web && npm run typecheck
	cd apps/web && npm run build
	cd apps/web && npm run test
	cd apps/web && npm audit --audit-level=high

verify-scripts:
	node --check scripts/dev.mjs
	node --check scripts/smoke-clearflow.mjs
	node --check scripts/use-local-env.mjs

compose-check:
	docker compose config >/dev/null

vuln:
	cd services/api && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
