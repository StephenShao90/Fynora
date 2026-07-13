# Security And Compliance Notes

Clearflow is a portfolio/backend project, not a regulated financial institution. It is designed to show production-minded fintech engineering patterns without storing bank credentials directly.

## Credential Boundaries

- Bank login is delegated to Plaid Link.
- Stripe account authorization is delegated to Stripe OAuth.
- Clearflow never asks users for bank usernames or passwords.
- Plaid and Stripe provider tokens stay server-side.
- Provider tokens are protected with AES-GCM through `PROVIDER_TOKEN_ENCRYPTION_KEY`.
- Production startup should fail if provider token encryption is missing.

## Data Stored

Clearflow stores:

- users and organization memberships
- payment, refund, fee, payout, and bank transaction records
- reconciliation runs, matches, exceptions, and notes
- portfolio accounts, holdings, portfolio transactions, import records, and row-level import errors
- provider connection metadata and encrypted provider tokens
- webhook event metadata and audit logs

Clearflow does not store:

- bank credentials
- full card numbers
- Plaid Link credentials
- raw Stripe secret keys in frontend code
- Plaid or Stripe provider tokens in API responses

## Auditability

Important financial actions emit structured logs and/or audit records:

- `stripe.sync.completed`
- `bank.sync.completed`
- `reconciliation.run.created`
- `portfolio.holdings_imported`
- `portfolio.transactions_imported`
- `plaid.investments_synced`
- `stripe.connected`
- `stripe.disconnected`
- webhook receipt and dedupe events

Every HTTP response includes `X-Request-ID`. Frontend logs include the same request ID so browser reports can be matched to API logs.

## Webhook Security

- Stripe webhooks verify `Stripe-Signature` when `STRIPE_WEBHOOK_SECRET` is configured.
- Plaid webhook verification can be required with `PLAID_WEBHOOK_VERIFICATION=true`.
- Development mock webhook bypass is available only outside production.
- Webhook events are deduped by provider event ID where available.

## Operational Controls

- Redis-backed rate limiting can be enabled with `REDIS_ENABLED=true`.
- Idempotency keys protect financial write routes from accidental replay.
- Background jobs use Postgres row locking so multiple workers can scale safely.
- `/ready` verifies the database is reachable and critical tables are present.

## Remaining Compliance Work Before Real Customers

- Legal terms, privacy policy, and data retention policy.
- Formal incident response process.
- Secrets rotation process.
- Access review for production infrastructure.
- SOC 2 style control mapping if selling to businesses.
- Pen test before handling real customer financial data at scale.
