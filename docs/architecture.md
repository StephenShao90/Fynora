# Architecture

Clearflow is organized as a monorepo with `apps/web` for the Next.js operations dashboard and `services/api` for the Go API.

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
  |-> Stripe-style processor sync
  |-> Payment, refund, fee, payout models
  |-> Bank transaction models
  |-> Reconciliation Engine
  |-> Exception Tracking
  |-> Cash-Flow Forecasting
  |-> PostgreSQL schema
```

The current MVP uses an in-process demo store so reviewers can run the app instantly. PostgreSQL migrations define the production persistence model. The highest-priority production upgrade is implementing repository methods for the Clearflow entities and wiring them into the handlers.

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
