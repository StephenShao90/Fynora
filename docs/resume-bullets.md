# Resume Bullets

## Backend

- Built a Go-based payment reconciliation API that ingests Stripe-style charges, refunds, fees, payouts, and bank transactions, then matches processor payouts to bank deposits using deterministic reconciliation rules.
- Designed financial operations APIs for organizations, payment sync, bank sync, reconciliation runs, exception management, cash-flow summaries, forecasts, Plaid bank connectivity, and audit logging.
- Implemented JWT authentication, encrypted Plaid access-token storage, structured logging, PostgreSQL migrations, Dockerized local development, GitHub Actions CI, and Go tests for production-style reliability.

## Full Stack

- Built Clearflow, a full-stack payment operations platform with Go, Next.js, TypeScript, PostgreSQL schema design, Plaid Link, and Recharts dashboards for small organizations.
- Developed a reconciliation dashboard that lets operators sync processor/bank data, run payout-to-deposit matching, review exceptions, resolve breaks, and monitor cash-flow forecasts.
- Created a polished demo flow for student clubs and small teams, showing matched payouts, refunds, processing fees, unmatched deposits, monthly reports, and cash balance projections.

## Fintech / Payments

- Built a payment reconciliation and cash-flow intelligence system that models core fintech concepts including payouts, settlement timing, refunds, processing fees, bank deposits, reconciliation breaks, and audit logs.
- Designed an extensible integration layer where mock Stripe sync can be replaced by real Stripe API/webhook ingestion and Plaid bank transactions can feed operational cash-flow analytics.
- Delivered a backend-heavy portfolio project aligned with payment infrastructure roles: APIs, data integrity, event-style ingestion, idempotent upserts, reconciliation algorithms, and financial exception workflows.
