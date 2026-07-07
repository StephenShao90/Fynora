# Deployment Guide

Clearflow intentionally keeps the frontend and backend as separate deployable surfaces. Vercel is a good public demo host for the Next.js app, but the Go API and worker should run on a backend platform when you want the full product online.

## Option 1: Vercel Demo Only

Best for a portfolio link.

- Deploy `apps/web` to Vercel.
- Do not configure `NEXT_PUBLIC_API_BASE_URL`.
- Do not add backend secrets to Vercel.
- The frontend uses intentional demo/fallback financial data.
- The UI shows sample payments, payouts, bank deposits, matches, anomalies, recommendations, jobs, metrics, and provider connection examples.

This mode is not pretending to be production. It is a public product walkthrough while the Go API remains a separate deployable service.

## Option 2: Full Production Architecture

Best for a real hosted product.

- Deploy `apps/web` to Vercel.
- Deploy the Go API to Render, Railway, Fly.io, AWS App Runner, ECS, or a similar backend host.
- Deploy the worker as a separate service/process using the same API image and `/app/worker` command.
- Use hosted Postgres.
- Use hosted Redis if `REDIS_ENABLED=true`.
- Apply all SQL migrations before release.
- Set Vercel `NEXT_PUBLIC_API_BASE_URL` to the deployed API URL.
- Set backend env vars on the backend host, not in Vercel.

Required backend env examples:

```bash
APP_ENV=production
DATABASE_URL=...
JWT_SECRET=...
ALLOWED_ORIGINS=https://your-vercel-app.vercel.app
PROVIDER_TOKEN_ENCRYPTION_KEY=...
PLAID_CLIENT_ID=...
PLAID_SECRET=...
PLAID_ENV=production
PLAID_WEBHOOK_VERIFICATION=true
STRIPE_CLIENT_ID=...
STRIPE_SECRET_KEY=...
STRIPE_WEBHOOK_SECRET=...
STRIPE_REDIRECT_URL=https://api.example.com/api/v1/integrations/stripe/callback
```

Optional production env:

```bash
REDIS_ENABLED=true
REDIS_URL=...
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=...
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_SAMPLE_RATIO=0.25
```

## Option 3: Local Full Stack

Best for technical demos and interviews.

```bash
cp .env.example .env
make install
make docker-up
make migrate
make api
```

In separate terminals:

```bash
make worker
make web
```

Open `http://localhost:3000`, click **Try Demo**, and walk through Dashboard, Reconciliation, Cash Flow, Ops, and Integrations.

## What Not To Do

- Do not move the Go API into Next.js API routes just to fit Vercel.
- Do not run the worker on Vercel serverless functions.
- Do not put Plaid, Stripe, database, JWT, Redis, or OTEL secrets into the frontend environment.
- Do not use Vercel demo mode as a substitute for the production backend architecture.
