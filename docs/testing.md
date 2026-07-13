# Testing Strategy

Clearflow uses three layers of verification.

## Static And Unit Gate

Run this before pushing changes:

```bash
make verify
```

This checks:

- Go formatting, unit tests, vet, and API/worker builds
- Next.js linting, TypeScript typechecking, production build, and Vitest
- high-severity npm audit
- Node script syntax
- Docker Compose configuration

## Backend Edge Cases Covered By Tests

The Go test suite covers the core backend risk areas:

- JWT signing and parsing
- configuration validation for production mode
- organization-scoped API access and authorization failures
- idempotency replay and request-body conflict behavior
- reconciliation matching and scoring logic
- payout explanation math
- cash-flow forecasting
- anomaly and recommendation logic
- Plaid and Stripe integration callback states
- webhook persistence, deduplication, and verification paths
- repository behavior for Postgres-backed features

## Live Smoke Gate

Run this with Postgres, Redis, API, worker, and web available:

```bash
make smoke
```

The smoke test verifies:

- health and auth
- demo organization bootstrap
- processor sync
- bank sync
- reconciliation run
- payout ledger and payout explanation
- cash-flow forecast
- anomalies and cash recommendations
- reconciliation match scoring
- job listing and async worker completion
- audit logs
- operational metrics
- idempotency replay

## Manual Browser Gate

Use [`docs/verification.md`](verification.md) for the exact browser flow and log lines to capture.
