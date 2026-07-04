# Fynora

Fynora is an AI-powered spending intelligence, personal finance planning, and portfolio management platform for students and young professionals. It connects daily spending, monthly cash flow, emergency fund planning, investing capacity, portfolio allocation, concentration risk, and grounded advisor-style explanations.

Fynora is an educational planning tool, not a registered financial advisor. It does not execute trades, scrape brokerages, store brokerage credentials, or recommend buying or selling specific individual stocks. Projections are hypothetical and not guaranteed.

## Why It Is More Than CRUD

Fynora includes transaction normalization, rule-based categorization, recurring subscription detection, duplicate charge checks, anomaly detection, emergency fund planning, monthly allocation recommendations, investment growth simulation, portfolio allocation analytics, concentration risk checks, mock market data, and deterministic advisor responses grounded in computed user data.

## Architecture

```text
User
  |
  v
Next.js Dashboard
  |
  v
Go API
  |------ PostgreSQL schema
  |------ Raw Event Store / S3-ready interface
  |------ Spending Intelligence Engine
  |------ Portfolio Analytics Engine
  |------ Market Data Provider
  |------ Optional AI Advisor API
```

## Tech Stack

- Frontend: Next.js App Router, TypeScript, React, Tailwind CSS, Recharts
- Backend: Go, layered internal packages, JWT auth, bcrypt password hashing
- Data: PostgreSQL migrations plus local demo store for frictionless portfolio demos
- Storage: local raw event store with S3-ready interface
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

Open `http://localhost:3000` and click **Try Demo**. The demo flow seeds a user, advisor profile, transactions, brokerage account, holdings, and portfolio transactions automatically.

## API Highlights

- `POST /auth/register`, `POST /auth/login`, `POST /auth/demo-token`, `GET /me`
- `POST /imports/transactions-csv`, `GET /transactions`
- `GET /insights/monthly-summary`, `/categories`, `/subscriptions`, `/anomalies`, `/cash-flow`
- `GET /advisor/plan`, `/emergency-fund`, `/account-priority`, `POST /advisor/investment-projection`, `POST /advisor/chat`
- `POST /portfolio/import/holdings-csv`, `GET /portfolio/summary`, `/allocation`, `/risk`, `/rebalance-suggestions`, `/projected-growth`
- `GET /market/quote/{symbol}`, `POST /market/quotes`

## Broker Integration Strategy

Fynora starts with manual holdings and CSV imports, including Wealthsimple-style holdings exports. Future real integrations should use official APIs or approved aggregators such as Plaid Investments, SnapTrade, or Flinks. The app should never ask for or store raw brokerage passwords.

## Sample Data

Sample CSVs live in `sample-data/`:

- `sample_transactions.csv`
- `sample_holdings.csv`
- `sample_portfolio_transactions.csv`

## Testing

```bash
make fmt
make test
make build
```

## Deployment Notes

The frontend can deploy to Vercel with `NEXT_PUBLIC_API_BASE_URL` pointing at the API. The Go API can be containerized for AWS ECS/App Runner or adapted for Lambda. PostgreSQL maps naturally to RDS, and raw imports can move from local storage to S3 behind the existing storage interface.

## Future Improvements

- Replace the demo memory store with full PostgreSQL repositories in all handlers
- Add `sqlc` generated queries and repository integration tests
- Add OAuth-style broker aggregator connections
- Persist portfolio snapshots for richer performance charts
- Add OpenAI-compatible advisor calls when `OPENAI_API_KEY` is configured
- Add frontend component tests and Playwright smoke tests
