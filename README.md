# Clearflow

Clearflow is a payment reconciliation and cash-flow intelligence API for small businesses, creators, student organizations, nonprofits, and event teams.

It connects payment processor activity, bank deposits, and manual or CSV imports so operators can answer questions they usually handle in spreadsheets:

- Which bank deposit maps to which Stripe payout?
- Which payments, fees, and refunds make up a payout?
- Which deposits or payouts are unmatched?
- How much cash is available now?
- What will cash look like over the next 7, 30, or 60 days?

Clearflow does not move money, store bank credentials, or scrape banks. Bank connectivity is delegated to providers such as Plaid; processor connectivity is designed around APIs and webhooks such as Stripe.

## Why This Is A Backend Portfolio Project

Clearflow is not a CRUD demo. The backend includes JWT auth, organization membership, Postgres-backed financial records, integer minor-unit money storage, idempotent ingestion, payout breakdowns, reconciliation matching, exception workflows, cash-flow reporting, audit logs, structured JSON logging, OpenAPI docs, Docker Compose, and a background worker pattern.

That makes it useful both as a product foundation and as a backend systems project for fintech-style roles.

## Why This Project Is Backend-Heavy

The product is intentionally built around backend/platform concerns that show up in fintech roles:

- Go API with explicit HTTP handlers and production-style middleware
- PostgreSQL persistence for users, organizations, financial records, jobs, idempotency keys, webhooks, audit logs, and provider connections
- JWT access tokens plus refresh-token sessions
- Organization RBAC for owner/admin/viewer workflows
- Idempotency keys and request-body hashing for financial writes
- Async jobs and a separate worker process for sync and reconciliation work
- Plaid/Stripe integration architecture without storing bank credentials
- Webhook verification, persistence, deduplication, and job queueing
- Audit logs, operational metrics, and OpenTelemetry-ready tracing
- Redis-ready rate limiting and in-flight idempotency locks
- Reconciliation scoring, payout explanations, anomaly detection, and cash-flow forecasting
- Frontend intelligence dashboard that exercises the backend instead of acting as a static mock

Resume bullet:

Built a production-style fintech operations platform in Go, PostgreSQL, Next.js, and TypeScript with organization RBAC, idempotent financial writes, async reconciliation jobs, Plaid/Stripe integration architecture, webhook processing, audit logs, metrics, OpenTelemetry-ready tracing, and financial intelligence APIs for payout explanation, anomaly detection, and cash-flow forecasting.

## Architecture

```text
Next.js operations dashboard
  |
  v
Go API
  |-- JWT auth
  |-- organization membership and RBAC-ready roles
  |-- idempotent processor/bank ingestion
  |-- reconciliation engine
  |-- cash-flow/reporting endpoints
  |
  v
PostgreSQL
  |-- organizations, memberships
  |-- payments, refunds, fees, payouts, payout_items
  |-- bank_transactions
  |-- reconciliation_runs, matches, exceptions
  |-- idempotency_keys, audit_logs, sync_jobs, webhook_events
```

The API keeps an in-memory fallback so the app can still be demoed if Postgres is unavailable, but the intended runtime path is PostgreSQL.

## Tech Stack

- Backend: Go, `net/http`, JWT, bcrypt, pgx, structured JSON logs
- Database: PostgreSQL 16 with SQL migrations
- Frontend: Next.js App Router, TypeScript, Tailwind CSS
- Integrations: Plaid Link scaffold, Stripe-style ingestion architecture
- DevOps: Docker Compose, Makefile, GitHub Actions CI, OpenAPI

## Local Setup

Install Go first if `go` is missing:

```bash
brew install go
```

Then run:

```bash
cp .env.example .env
make install
make dev
```

The Docker Postgres container is exposed on host port `5433` to avoid colliding with a local Postgres install. If you already have a `.env`, make sure `DATABASE_URL=postgres://postgres:postgres@localhost:5433/clearflow?sslmode=disable`.

Open `http://localhost:3000` and click **Try Demo**. The demo seeds a user, organization, Stripe-style payments, fees, refund, payout, bank transactions, payout item breakdown, and reconciliation run.

To start the full stack, run the smoke suite, and shut it down automatically:

```bash
make dev-smoke
```

## Useful API Flow

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/demo-token | jq -r .token)

curl -s http://localhost:8080/debug/clearflow \
  -H "Authorization: Bearer $TOKEN" | jq

curl -s -X POST http://localhost:8080/sync/stripe \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: stripe-demo-1" | jq

curl -s -X POST http://localhost:8080/sync/bank \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: bank-demo-1" | jq

curl -s -X POST http://localhost:8080/reconciliation/runs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: recon-demo-1" | jq
```

API contract: [`docs/openapi.yaml`](docs/openapi.yaml)

Recruiter demo script: [`docs/recruiter-demo.md`](docs/recruiter-demo.md)

Main feature video demo: [`docs/assets/clearflow-main-feature-demo.webm`](docs/assets/clearflow-main-feature-demo.webm)

Code organization guide: [`docs/code-organization.md`](docs/code-organization.md)

Performance model: [`docs/performance.md`](docs/performance.md)

## Demo Flow

1. Start Postgres, Redis, migrations, API, worker, and web with `make dev`.
2. Open `http://localhost:3000`, click **Try Demo**, and land on the operations dashboard.
3. Open **Onboarding** or **Dashboard** and click **Run full demo setup** to seed onboarding, processor data, bank data, reconciliation, and portfolio sample data in order.
4. Open **Reconciliation** and review the match rate, exception queue, payout ledger, payout explanation, and exception workbench.
5. Resolve a break with an operator note and confirm the note history persists.
6. Open **Transactions** to search/filter the ledger and update categories.
7. Open **Cash Flow** to show forecasts, anomalies, recommendations, and match scoring.
8. Open **Ops** to show async jobs, audit logs, metrics, idempotency, Redis readiness, and tracing readiness.
9. Open **Integrations** to show Stripe/Plaid connection state, provider sync controls, and webhook security notes.
10. Open **Settings** to show team roles, sessions, demo reset, and production deployment separation.

Interview-ready backend summary:

Built a production-style fintech API with JWT auth, refresh-token sessions, organization RBAC, Postgres-backed financial data, idempotent write routes, async sync/reconciliation jobs, Plaid persistence, webhook handling, audit logs, metrics, and financial intelligence endpoints for reconciliation scoring, payout explanations, anomaly detection, cash recommendations, spending insights, and cash-flow forecasting.

## Core Endpoints

- Auth: `POST /auth/register`, `POST /auth/login`, `POST /auth/demo-token`, `GET /me`
- Organizations: `POST /organizations`, `GET /organizations`
- Operations data: `GET /payments`, `GET /payouts`, `GET /payouts/{id}/breakdown`, `GET /bank-transactions`
- Ingestion: `POST /sync/stripe`, `POST /sync/bank`
- Reconciliation: `POST /reconciliation/runs`, `GET /reconciliation/runs`, `GET /reconciliation/runs/{id}`, `GET /reconciliation/exceptions`, `PATCH /reconciliation/exceptions/{id}`
- Cash flow: `GET /cash-flow/summary`, `GET /cash-flow/forecast`, `GET /api/v1/cashflow/forecast`, `GET /reports/monthly`
- Intelligence: `GET /api/v1/payouts/{id}/explanation`, `GET /api/v1/insights/anomalies`, `GET /api/v1/insights/spending`, `GET /api/v1/recommendations/cash`, `GET /api/v1/reconciliation-runs/{id}/matches`
- Plaid scaffold: `POST /connections/plaid/link-token`, `POST /connections/plaid/exchange-public-token`, `POST /connections/plaid/sync-transactions`
- Debugging: `GET /health`, `GET /ready`, `GET /debug/clearflow`

## Plaid

Add Plaid keys to `.env`:

```bash
PLAID_CLIENT_ID=...
PLAID_SECRET=...
PLAID_ENV=sandbox
PLAID_PRODUCTS=transactions
PLAID_COUNTRY_CODES=US,CA
PLAID_WEBHOOK_VERIFICATION=false
STRIPE_CLIENT_ID=...
STRIPE_SECRET_KEY=...
STRIPE_WEBHOOK_SECRET=...
STRIPE_REDIRECT_URL=http://localhost:8080/api/v1/integrations/stripe/callback
PROVIDER_TOKEN_ENCRYPTION_KEY=...
REDIS_ENABLED=false
REDIS_URL=redis://localhost:6379/0
REDIS_TLS=false
OTEL_ENABLED=false
OTEL_SERVICE_NAME=clearflow-api
OTEL_EXPORTER_OTLP_ENDPOINT=
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_HEADERS=
OTEL_SAMPLE_RATIO=1.0
OTEL_ENVIRONMENT=development
```

Plaid handles bank authentication. Clearflow exchanges the public token server-side, protects provider tokens with the configured token protector, and never returns raw Plaid or Stripe tokens from status endpoints.

## Stripe and Webhook Hardening

Phase 7 adds an OAuth-ready Stripe integration shape:

- `GET /api/v1/integrations/stripe/connect-url` creates an expiring OAuth state and Stripe Connect URL.
- `GET /api/v1/integrations/stripe/callback` validates state, protects provider tokens, stores account metadata, audits the connection, and emits an outbox event.
- `GET /api/v1/integrations/stripe/status` returns safe connection metadata only.
- `DELETE /api/v1/integrations/stripe` marks the local connection disconnected and emits an outbox event.
- `POST /api/v1/webhooks/processors/stripe` verifies `Stripe-Signature`, persists and dedupes events, and queues relevant sync jobs.

Provider tokens are protected with `PROVIDER_TOKEN_ENCRYPTION_KEY`. Production mode fails fast if token encryption is missing or short, demo auth is enabled, CORS is wildcarded, Plaid is not in production mode, Plaid webhook verification is disabled, or Stripe webhook secrets are missing. Development mock webhook bypass is only allowed outside production.

## Production Reliability

Phase 8 adds optional Redis-backed rate limiting and in-flight idempotency locks. Redis is disabled by default for local development. Set `REDIS_ENABLED=true` and `REDIS_URL=...` to use Redis counters/locks; production fails fast if Redis is enabled but unavailable.

OpenTelemetry tracing can be enabled with `OTEL_ENABLED=true`. The API and worker support OTLP export over `grpc` or `http/protobuf`, optional OTLP headers, and configurable sampling through `OTEL_SAMPLE_RATIO`. Request logs and worker job logs include `trace_id` and `span_id`; trace context is propagated through HTTP, webhooks, queued jobs, and financial intelligence operations. Local development does not require a collector, while production fails fast if tracing is enabled without a valid endpoint.

Production architecture summary:

Clearflow is a production-style fintech operations platform built with Go, PostgreSQL, JWT/session auth, organization RBAC, idempotent financial writes, async job workers, provider webhook ingestion, Plaid/Stripe integration hardening, audit logs, metrics, OpenTelemetry tracing, Redis-backed rate limiting, and financial intelligence APIs for reconciliation scoring, payout explanations, anomalies, forecasts, and cash recommendations.

## Testing

```bash
make verify
```

`make verify` runs backend formatting, unit tests, vetting, API/worker builds, frontend linting, typechecking, production build, Vitest, high-severity npm audit, Node script syntax checks, and Docker Compose config validation.

For live end-to-end verification with the API, worker, Postgres, and Redis running:

```bash
make smoke
```

Or use `make dev-smoke` to start the local stack, run smoke verification, and stop the app in one command.

To regenerate the recorded product walkthrough while the frontend is running:

```bash
make record-demo
```

Detailed verification notes live in [`docs/verification.md`](docs/verification.md). Customer launch gates live in [`docs/production-readiness.md`](docs/production-readiness.md). Do not claim real customer volume, production throughput, uptime, or deployed usage until those claims are backed by telemetry.

## Deployment

Clearflow supports three deployment modes:

- **Mode A: Vercel demo mode.** Deploy `apps/web` to Vercel without backend secrets. If `NEXT_PUBLIC_API_BASE_URL` is not configured, the frontend uses intentional sample financial data and shows a demo-mode indicator.
- **Mode B: Full local stack.** Run frontend, Go API, worker, Postgres, and Redis locally with `make dev`.
- **Mode C: Real production architecture later.** Keep the frontend on Vercel, deploy the Go API and worker to Render, Railway, Fly.io, AWS App Runner, ECS, or a similar backend host, and point Vercel `NEXT_PUBLIC_API_BASE_URL` at that API.

Do not move the Go API or worker into Next.js API routes just to fit Vercel. The backend is intentionally a separate service.

## Resume Summary

Built Clearflow, a Go and PostgreSQL payment reconciliation API that ingests processor payouts and bank transactions, stores money in integer minor units, enforces organization-scoped access, makes sync operations idempotent, matches payouts to deposits, creates reconciliation exceptions, exposes payout breakdown and cash-flow reporting endpoints, and ships with OpenAPI, Docker, worker, CI, structured logs, and a Next.js operations dashboard.
