# Demo Walkthrough

Clearflow is a production-style fintech operations platform for small organizations. It reconciles processor payouts with bank deposits, explains payout composition, surfaces anomalies, forecasts cash flow, manages integrations, runs async jobs, and exposes operational audit/metrics/debug surfaces.

## Two To Three Minute Script

1. **Log in or use demo mode.** On local full stack, click **Try Demo**. On Vercel without `NEXT_PUBLIC_API_BASE_URL`, the UI intentionally shows sample financial data and a demo-mode badge.
2. **Open Onboarding.** Pick a demo company, save the workspace profile, and use the checklist to explain the customer setup path.
3. **View Dashboard.** Point out operating cash, net cash flow, processor fees, refunds, open breaks, forecast chart, payout volume, and recent processor/bank ledger rows.
4. **Open Reconciliation.** Run processor sync, bank sync, and reconciliation. Explain that financial write endpoints use idempotency keys and queue worker jobs.
5. **Resolve a break.** Open the exception workbench, add a resolution note, optionally associate a bank record, and resolve the break.
6. **Open payout explanation.** Click **View explanation** and walk through gross payments, fees, refunds, net deposit, linked bank deposit, warnings, and plain-English summary.
7. **Open Transactions.** Search/filter the ledger, select a transaction, inspect details, and update category.
8. **View cash-flow intelligence.** Open **Cash Flow**, change horizon, and explain the labeled forecast axes, assumptions, anomalies, recommendations, and match confidence.
9. **Open Ops.** Show async jobs, audit log entries, metrics, idempotency counters, Redis readiness, and OpenTelemetry trace readiness.
10. **Open Integrations and Settings.** Show Stripe/Plaid status, provider sync controls, team roles, sessions, and production deployment separation.

## Interview Talking Points

- The Go API is the product core; the frontend is an operations dashboard over real API contracts.
- PostgreSQL stores money in integer minor units for durable financial records.
- Organization RBAC scopes access to financial operations.
- Idempotency protects financial writes from duplicate retries.
- Sync/reconciliation work can run asynchronously in a separate worker.
- Plaid/Stripe integration architecture avoids storing bank credentials and verifies webhooks.
- Audit logs and metrics make operational debugging possible.
- Redis and OpenTelemetry are optional locally but ready for production hardening.

## Final Local Smoke Checklist

- [ ] Postgres starts
- [ ] Redis starts
- [ ] API starts
- [ ] Worker starts
- [ ] Web starts
- [ ] Demo login works
- [ ] Stripe mock sync works
- [ ] Bank sync works
- [ ] Reconciliation run creates job
- [ ] Worker processes job
- [ ] Payout explanation loads
- [ ] Anomalies load
- [ ] Forecast loads
- [ ] Recommendations load
- [ ] Jobs page loads
- [ ] Audit logs load
- [ ] Metrics endpoint works
- [ ] Logs include `trace_id`/`span_id` when tracing enabled
- [ ] Vercel frontend build passes

## Demo Modes

- **Vercel demo mode:** frontend only, sample financial data, no backend secrets required.
- **Full local stack:** frontend, Go API, worker, Postgres, and Redis through `make dev`.
- **Future production architecture:** frontend on Vercel, Go API and worker on a backend host, hosted Postgres/Redis, and `NEXT_PUBLIC_API_BASE_URL` pointing at the deployed API.
