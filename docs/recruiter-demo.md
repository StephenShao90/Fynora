# Recruiter Demo Guide

Clearflow is best presented as a backend-heavy fintech operations platform, not a generic finance dashboard.

## One-Sentence Pitch

Clearflow reconciles Stripe payouts to bank deposits, explains cash movement, preserves an audit trail for exceptions, and forecasts operating cash for small teams.

## Demo Path

1. Start the stack with `make dev`.
2. Open `http://localhost:3000`.
3. Click **Try Demo**.
4. Open **Onboarding** and show persisted setup status.
5. Open **Dashboard** and explain cash, fees, refunds, open breaks, and forecast.
6. Open **Reconciliation** and run the full workflow.
7. Open an exception, add an investigation note, resolve it, and show note history.
8. Click **View explanation** on a payout.
9. Open **Transactions**, search/filter, and update a category.
10. Open **Integrations**, run sync controls, and send sandbox webhook tests.
11. Open **Ops** and show **Bank-grade control evidence**: worker health, idempotency replay evidence, auditability, provider event handling, and job durability.
12. Open **Settings** and show team/RBAC and sessions.

## Backend Points To Say Out Loud

- The API is written in Go with PostgreSQL persistence and explicit SQL migrations.
- Money is stored in integer minor units in the Clearflow tables.
- Financial writes use idempotency keys.
- Sync/reconciliation work can run asynchronously through a worker and `sync_jobs`.
- Webhooks are persisted and deduped before queueing work.
- Stripe and Plaid credentials stay server-side.
- Reconciliation exceptions have operator notes and audit logs.
- Ops converts raw metrics/jobs/audit logs into a control-evidence view, which is how real financial platforms prove reliability and investigate incidents.
- The frontend is a client over real API contracts, not the source of truth.

## What To Show In Logs

- API: `database.connected`, `http.request`, `clearflow.operation`
- Worker: `worker.started`, `worker.job.started`, `worker.job.completed`
- Smoke: `19 passed, 0 failed`
- Browser: `[clearflow-api]` request objects with request IDs

## Product Boundary

The commercial wedge is Stripe plus bank reconciliation for small operators. Portfolio/advisor features are useful resume depth, but they are not the core product.

## What Is Still External

- Real Stripe production onboarding and live webhook dashboard setup.
- Real Plaid production approval and institution coverage.
- Hosted backend API, worker, Postgres, and Redis for a public full-stack demo.
- Compliance review before handling real customer financial workflows.
