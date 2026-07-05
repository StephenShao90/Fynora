# Clearflow

Clearflow is a payment reconciliation and cash-flow intelligence platform for small organizations: student clubs, creators, campus businesses, nonprofits, local event teams, tutors, and small service businesses.

It connects payment processor data, bank deposits, and optional CSV/manual imports, then answers the operational questions that small teams usually handle in spreadsheets:

- Which bank deposit came from which Stripe payout?
- Which charges, refunds, and fees are inside that payout?
- Which deposits are unmatched?
- Which payments failed or need follow-up?
- How much cash is available now?
- What will cash look like over the next 7, 30, or 60 days?

Clearflow does not move money, execute trades, store bank credentials, or scrape banks. It uses secure providers such as Plaid for bank connectivity and is designed to integrate with processors such as Stripe.

## Why It Is More Than CRUD

Clearflow includes processor sync, bank transaction ingestion, payout/deposit matching, exception generation, cash-flow forecasting, audit logs, idempotent-style upserts, Plaid Link integration, and a dashboard built around finance operations workflows. The core value is a reconciliation engine, not a table editor.

## Architecture

```text
Small organization
  |
  v
Next.js Operations Dashboard
  |
  v
Go API
  |------ PostgreSQL schema
  |------ Plaid bank connection
  |------ Stripe-style payment/payout sync
  |------ Reconciliation Engine
  |------ Exception Tracking
  |------ Cash-Flow Forecasting
  |------ Audit Logs
```

## Tech Stack

- Frontend: Next.js App Router, TypeScript, React, Tailwind CSS, Recharts
- Backend: Go, JWT auth, bcrypt password hashing, structured JSON logging
- Data: PostgreSQL migrations plus local demo store for frictionless evaluation
- Integrations: Plaid Link for secure bank connectivity; Stripe-style mock sync ready to replace with real Stripe APIs/webhooks
- DevOps: Docker Compose, Makefile, GitHub Actions CI

## Local Setup

```bash
cp .env.example .env
make install
make docker-up
make migrate
make api
make web
```

Open `http://localhost:3000` and click **Try Demo**. The demo flow seeds a user, organization, Stripe-style payments, fees, refund, payout, bank deposits, and a reconciliation run.

## Plaid Bank Data

Add your Plaid keys to `.env`:

```bash
PLAID_CLIENT_ID=...
PLAID_SECRET=...
PLAID_ENV=production
PLAID_PRODUCTS=transactions
PLAID_COUNTRY_CODES=US,CA
```

Run `make api` from the repo root so the Makefile exports `.env`. In the app, go to **Imports** and click **Connect bank with Plaid**. Plaid handles bank authentication; Clearflow receives a temporary public token, exchanges it server-side, encrypts the Plaid access token using `JWT_SECRET`, and stores it under `data/plaid-connections.json`.

Do not commit `.env` or the `data/` directory.

## API Highlights

- Auth: `POST /auth/register`, `POST /auth/login`, `POST /auth/demo-token`, `GET /me`
- Organizations: `POST /organizations`, `GET /organizations`
- Processor/bank sync: `POST /sync/stripe`, `POST /sync/bank`
- Reconciliation: `POST /reconciliation/runs`, `GET /reconciliation/runs`, `GET /reconciliation/runs/{id}`, `GET /reconciliation/exceptions`, `PATCH /reconciliation/exceptions/{id}`
- Cash flow: `GET /cash-flow/summary`, `GET /cash-flow/forecast`, `GET /reports/monthly`
- Operations data: `GET /payments`, `GET /payouts`, `GET /bank-transactions`
- Plaid: `POST /connections/plaid/link-token`, `POST /connections/plaid/exchange-public-token`, `POST /connections/plaid/sync-transactions`, `GET /connections`

## Demo Workflow

1. Click **Try Demo**.
2. Open **Reconciliation**.
3. Review the seeded payout/deposit match and unmatched deposit exception.
4. Click **Sync Stripe sample**, **Sync bank sample**, and **Run reconciliation** to generate another run.
5. Open **Dashboard** to see cash balance, fees/refunds, exceptions, payout chart, and cash forecast.

## Product Direction

The next production step is replacing `POST /sync/stripe` mock data with real Stripe OAuth/API sync and webhook ingestion:

- `payment_intent.succeeded`
- `charge.refunded`
- `payout.paid`
- `payout.failed`
- `balance.available`

The second production step is replacing the local memory store with PostgreSQL repositories for all Clearflow resources.

## Testing

```bash
make fmt
make test
make build
```

## Deployment Notes

The frontend can deploy to Vercel with `NEXT_PUBLIC_API_BASE_URL` pointing at the API. The Go API can run on AWS ECS/App Runner or Lambda. PostgreSQL maps naturally to RDS. Plaid secrets and future Stripe secrets should live in a managed secret store, not in the repo.

## Resume Summary

Built a Go-based payment reconciliation platform that ingests Stripe-style payouts and bank transactions, matches deposits to payments/refunds/fees, flags reconciliation exceptions, forecasts cash flow, and exposes a production-style API with auth, audit logs, Docker, CI, and a polished Next.js dashboard.
