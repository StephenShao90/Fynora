# Production Readiness

Clearflow is built to demonstrate real fintech backend patterns: tenant-scoped RBAC, provider-token encryption, signed webhook ingestion, idempotent financial writes, durable jobs, audit logs, and request tracing.

## Current Customer Gate

Do not onboard real merchants until every item below is complete and verified in the deployed environment:

- `APP_ENV=production` boots successfully with Postgres storage and no memory fallback.
- `ENABLE_DEMO_AUTH=false`; `POST /api/v1/auth/demo-token` returns 404.
- `ALLOWED_ORIGINS` is the exact HTTPS frontend origin.
- `PLAID_ENV=production` and `PLAID_WEBHOOK_VERIFICATION=true`.
- `STRIPE_WEBHOOK_SECRET` is configured and unsigned Stripe webhooks are rejected.
- `PROVIDER_TOKEN_ENCRYPTION_KEY` and `JWT_SECRET` are at least 32 characters and stored only in the backend host.
- Migrations are applied through `006_product_readiness.sql`.
- API, worker, and frontend are deployed as separate services.
- Smoke tests pass against the deployed API.
- Backups, restore testing, alerting, and incident contacts exist.
- Legal terms, privacy policy, and data deletion flow are published.

## Known Launch Blockers

- Browser auth currently uses bearer tokens from browser storage. For production merchants, move to HttpOnly Secure SameSite cookies or another hardened session model before handling real financial data.
- Stripe Connect currently uses OAuth-style onboarding. Confirm the target Stripe platform model and migrate to Accounts v2 if required for the live business.
- Any resume or sales claim about customer count, production usage, uptime, or requests per second must be backed by real telemetry.

## Evidence To Keep

Before a customer pilot, save:

- `make verify` output.
- deployed `/ready` response.
- deployed smoke-test output.
- webhook rejection tests for bad Stripe signatures and unsupported processors.
- screenshots of audit logs after Stripe sync, bank sync, reconciliation, exception resolution, and portfolio import.
- backup restore drill notes.
