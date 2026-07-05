# API

All protected endpoints require `Authorization: Bearer <token>`.

## Health

- `GET /health`

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
- `GET /bank-transactions`
- `POST /sync/stripe`
- `POST /sync/bank`

`POST /sync/stripe` currently loads Stripe-style demo payments, refunds, fees, and payouts. It is intentionally shaped so it can be replaced by real Stripe API/webhook ingestion.

`POST /sync/bank` currently loads demo bank deposits. Real bank data can be connected through Plaid.

## Reconciliation

- `POST /reconciliation/runs`
- `GET /reconciliation/runs`
- `GET /reconciliation/runs/{id}`
- `GET /reconciliation/exceptions`
- `PATCH /reconciliation/exceptions/{id}`

The reconciliation engine matches processor payouts to bank deposits by amount, arrival date, and description. It creates exceptions for unmatched payouts and unmatched deposits.

## Cash Flow

- `GET /cash-flow/summary`
- `GET /cash-flow/forecast`
- `GET /reports/monthly`

## Plaid Connections

- `GET /connections`
- `DELETE /connections/{id}`
- `POST /connections/plaid/link-token`
- `POST /connections/plaid/exchange-public-token`
- `POST /connections/plaid/sync-transactions`

## Legacy Personal-Finance Endpoints

The codebase still includes Fynora-era personal-finance endpoints for transaction analysis, advisor projections, and portfolio analytics. These remain useful reference modules but are secondary to the Clearflow product direction.
