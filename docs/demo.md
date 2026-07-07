# Demo Walkthrough

Clearflow is a production-style fintech operations platform for small organizations. It reconciles processor payouts with bank deposits, explains payout composition, surfaces anomalies, forecasts cash flow, manages integrations, runs async jobs, and exposes operational audit/metrics/debug surfaces.

## Two To Three Minute Script

1. **Log in or use demo mode.** On local full stack, click **Try Demo**. On Vercel without `NEXT_PUBLIC_API_BASE_URL`, the UI intentionally shows sample financial data and a demo-mode badge.
2. **View Dashboard.** Point out operating cash, net cash flow, processor fees, refunds, open breaks, forecast chart, payout volume, and recent processor/bank ledger rows.
3. **Open Reconciliation.** Run processor sync, bank sync, and reconciliation. Explain that real financial write endpoints support idempotency keys and queue work for the worker.
4. **Explain exact and likely matches.** Show the exact payout/deposit match, the likely amount-mismatch match, the missing payout, and the unmatched deposit.
5. **Open payout explanation.** Click **View explanation** and walk through gross payments, fees, refunds, net deposit, linked bank deposit, warnings, and plain-English summary.
6. **View anomalies.** Open **Cash Flow** and show missing payout, unmatched deposit, and high-fee anomaly examples.
7. **View cash-flow forecast.** Change the horizon and explain assumptions, confidence, projected balance, inflows, and outflows.
8. **View recommendations.** Show the reserve recommendation, missing payout follow-up, and fee review recommendation.
9. **Open Ops.** Show async jobs, audit log entries, metrics, idempotency counters, Redis readiness, and OpenTelemetry trace readiness.
10. **Open Integrations.** Show Stripe and Plaid connection status, Stripe Connect flow shape, webhook verification notes, and the fact that bank credentials are delegated to Plaid.

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
- **Full local stack:** frontend, Go API, worker, Postgres, and Redis.
- **Future production architecture:** frontend on Vercel, Go API and worker on a backend host, hosted Postgres/Redis, and `NEXT_PUBLIC_API_BASE_URL` pointing at the deployed API.
