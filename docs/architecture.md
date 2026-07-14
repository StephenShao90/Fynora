# Architecture

Clearflow is organized as a monorepo with `apps/web` for the Next.js operations dashboard and `services/api` for the Go API.

The product wedge is intentionally narrow: Stripe-style processor activity plus Plaid/bank activity for payout reconciliation and cash visibility. Portfolio/advisor modules remain useful technical extensions, but the core business workflow is onboarding, provider ingestion, reconciliation, exception review, auditability, and forecasted operating cash.

```text
Small organization
  |
  v
Next.js Dashboard
  |
  v
Go API
  |-> Auth and organizations
  |-> Plaid bank connection
  |-> Plaid Investments-shaped portfolio sync
  |-> Stripe-style processor sync
  |-> Payment, refund, fee, payout models
  |-> Bank transaction models
  |-> Reconciliation Engine
  |-> Exception Tracking
  |-> Cash-Flow Forecasting
  |-> PostgreSQL schema
```

The current MVP uses an in-process demo store so reviewers can run the app instantly. PostgreSQL migrations define the production persistence model. The highest-priority production upgrade is implementing repository methods for the Clearflow entities and wiring them into the handlers.
The production path now uses PostgreSQL through `internal/repository.ClearflowRepository`. The in-process store remains as a fallback for quick demos when `DATABASE_URL` is unavailable.

## Persistence Model

Clearflow stores financial data in integer minor units:

- `payments.amount_minor`
- `refunds.amount_minor`
- `fees.amount_minor`
- `payouts.amount_minor`
- `payout_items.amount_minor`
- `bank_transactions.amount_minor`
- `reconciliation_matches.amount_minor`

The public JSON API still returns major-unit decimal amounts for frontend compatibility.

Organization access is represented by `organization_members` with roles: `owner`, `admin`, `analyst`, and `viewer`. Current handlers enforce organization membership; the role column is ready for deeper RBAC policy checks.

Idempotent writes use `idempotency_keys` scoped by `(user_id, key)`. Reusing the same key and request hash replays the stored response. Reusing the key with a different request returns `409 IDEMPOTENCY_CONFLICT`.

Background sync work is modeled with `sync_jobs`; the worker process claims queued jobs with `FOR UPDATE SKIP LOCKED` so multiple workers can scale safely.

## Reconciliation Flow

```text
Stripe charges/refunds/fees/payouts
        +
Plaid bank deposits
        |
        v
Amount/date/description matching
        |
        +--> reconciliation_matches
        |
        +--> reconciliation_exceptions
```

The matching engine produces explanations and confidence scores so the dashboard can show why a payout matched a bank deposit or why an exception needs review.

## Financial Intelligence Layer

The Phase 5 and Phase 6 intelligence layer turns stored financial records into operator-facing answers:

- Payout explanations answer why a deposit happened by showing gross payments, fees, refunds, net amount, matching bank deposit, warnings, and a plain-English summary.
- Reconciliation match intelligence scores payout-to-deposit candidates and returns match status, confidence, reasons, amount difference, and explanation.
- Cash-flow forecasting projects daily balances across 7, 30, 60, or 90 days with assumptions and confidence.
- Anomaly detection surfaces missing payouts, delayed payouts, unmatched deposits, elevated processor fees, and high refund activity.
- Spending insights summarize debit activity by category, percentage, merchant, and notes.
- Cash recommendations turn forecast and anomaly signals into operational next steps without making regulated investment claims.

## Integration Boundaries

- Plaid owns bank authentication and returns transaction data to Clearflow.
- Portfolio ingestion uses a normalized ledger for CSV imports and Plaid Investments-shaped sync results. The current investment sync endpoint imports deterministic mock data into the same durable tables; the boundary is ready for real Plaid Investments holdings and investment transactions.
- Stripe-style processor ingestion currently uses deterministic sample data, but the database includes `processor_accounts` and `webhook_events` for real API/webhook ingestion.
- Clearflow stores reconciliation state, exceptions, audit logs, payout breakdowns, and reporting aggregates.

## Provider Integration Security

Phase 7 adds a production-ready integration shape:

- Stripe Connect URLs use expiring OAuth states stored as hashes with organization and user context.
- Stripe callbacks validate state reuse/expiry, protect provider tokens, and persist only connection metadata in status responses.
- Stripe webhook ingestion verifies `Stripe-Signature` when `STRIPE_WEBHOOK_SECRET` is configured, dedupes by Stripe event id, persists events, queues supported sync jobs, audits receipt, and emits outbox events.
- Plaid webhook verification is isolated behind `PlaidWebhookVerifier`; `PLAID_WEBHOOK_VERIFICATION=true` requires verification and development mock bypass is only available outside production.
- Provider token protection uses AES-GCM when `PROVIDER_TOKEN_ENCRYPTION_KEY` is set. Production startup fails if provider token encryption is missing.
- Redis-backed rate limiting and in-flight idempotency locks can be enabled with `REDIS_ENABLED=true` while Postgres remains the durable idempotency source of truth.
- OpenTelemetry-style tracing can be enabled with `OTEL_ENABLED=true`; request logs include trace IDs and metrics include trace-start counters.

Required integration variables:

- `STRIPE_CLIENT_ID`
- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `STRIPE_REDIRECT_URL`
- `PROVIDER_TOKEN_ENCRYPTION_KEY`
- `PLAID_WEBHOOK_VERIFICATION`
- `REDIS_ENABLED`
- `REDIS_URL`
- `REDIS_TLS`
- `OTEL_ENABLED`
- `OTEL_SERVICE_NAME`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_ENVIRONMENT`
