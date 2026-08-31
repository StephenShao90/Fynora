# API

All protected endpoints require `Authorization: Bearer <token>`.

## Health

- `GET /health`
- `GET /ready`

## Auth

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/demo-token`
- `GET /me`

## Organizations

- `POST /organizations`
- `GET /organizations`

## Payment Operations

- `GET /payments`
- `GET /payouts`
- `GET /payouts/{id}/breakdown`
- `GET /bank-transactions`
- `POST /sync/stripe`
- `POST /sync/bank`

`POST /sync/stripe` currently loads Stripe-style demo payments, refunds, fees, and payouts. It is intentionally shaped so it can be replaced by real Stripe API/webhook ingestion.

`POST /sync/bank` currently loads demo bank deposits. Real bank data can be connected through Plaid.

`POST /sync/stripe`, `POST /sync/bank`, and `POST /reconciliation/runs` support `Idempotency-Key`. Same key plus same request replays the stored JSON response; same key plus different request returns `409`.

## Reconciliation

- `POST /reconciliation/runs`
- `GET /reconciliation/runs`
- `GET /reconciliation/runs/{id}`
- `GET /reconciliation/exceptions`
- `PATCH /reconciliation/exceptions/{id}`
- `GET /reconciliation/exceptions/{id}/notes`
- `POST /reconciliation/exceptions/{id}/notes`

The reconciliation engine matches processor payouts to bank deposits by amount, arrival date, and description. It creates exceptions for unmatched payouts and unmatched deposits. Operators can resolve exceptions with notes or append investigation notes without resolving; notes persist in `exception_notes` and are reflected in audit logs.

## Onboarding

- `GET /api/v1/onboarding/status`
- `PUT /api/v1/onboarding/status`

Onboarding status persists selected demo/product scenario, business type, checklist state, and derived provider readiness. Provider readiness is computed from actual workspace, Stripe/Plaid connection, processor data, bank data, team, and open-break state.

## Cash Flow

- `GET /cash-flow/summary`
- `GET /cash-flow/forecast`
- `GET /reports/monthly`

## OpenAPI

See `docs/openapi.yaml`.

## Plaid Connections

- `GET /connections`
- `DELETE /connections/{id}`
- `POST /connections/plaid/link-token`
- `POST /connections/plaid/exchange-public-token`
- `POST /connections/plaid/sync-transactions`
- `POST /connections/plaid/sync-investments`

`POST /connections/plaid/sync-investments` currently imports a Plaid-shaped mock investments payload into the same persistent portfolio ledger used by CSV imports. It is intentionally shaped so the implementation can be replaced with Plaid Investments holdings and investment transaction endpoints without changing the portfolio analytics layer.

## Portfolio

- `POST /portfolio/import/holdings-csv`
- `POST /portfolio/import/transactions-csv`
- `GET /portfolio/imports`
- `GET /portfolio/imports/{id}/errors`
- `POST /portfolio/holdings`
- `GET /portfolio/holdings`
- `GET /portfolio/transactions`
- `GET /portfolio/summary`
- `GET /portfolio/allocation`
- `GET /portfolio/performance`
- `GET /portfolio/risk`

Portfolio CSV imports return row-level `errors` with row number, field, code, message, and raw row data. When Postgres is configured, accounts, imports, holdings, portfolio transactions, and import errors persist across API restarts.

## Integrations

- `GET /api/v1/integrations/stripe/connect-url`
- `GET /api/v1/integrations/stripe/callback`
- `GET /api/v1/integrations/stripe/status`
- `DELETE /api/v1/integrations/stripe`
- `POST /api/v1/webhooks/processors/stripe`
- `POST /api/v1/webhooks/plaid`

Stripe OAuth creates a signed, expiring state tied to the organization and user. The callback validates the state, protects provider tokens server-side, stores account metadata, writes audit logs, and emits outbox events. Stripe status never returns raw access or refresh tokens.

Stripe webhooks verify `Stripe-Signature`; production startup requires `STRIPE_WEBHOOK_SECRET`. Plaid webhook verification is required in production with `PLAID_WEBHOOK_VERIFICATION=true`; development mock bypass is available outside production only. Production processor webhooks reject unsupported providers and require `organizationId`.

The Integrations UI includes local sandbox webhook testers. These are for development verification only; production webhook verification should be tested through Stripe/Plaid dashboards or signed webhook tooling.

## Legacy Personal-Finance Endpoints

The codebase still includes legacy personal-finance endpoints for transaction analysis, advisor projections, and portfolio analytics. These remain useful reference modules but are secondary to the Clearflow product direction.
