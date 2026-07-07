# Production Runbook

## Required Environment

- `APP_ENV=production`
- `DATABASE_URL`
- `JWT_SECRET`
- `ALLOWED_ORIGINS`
- `PROVIDER_TOKEN_ENCRYPTION_KEY`
- `STRIPE_CLIENT_ID`
- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `STRIPE_REDIRECT_URL`
- `PLAID_CLIENT_ID`
- `PLAID_SECRET`
- `PLAID_ENV`
- `PLAID_WEBHOOK_VERIFICATION=true`

## Optional Reliability

- `REDIS_ENABLED=true`
- `REDIS_URL=redis://...`
- `REDIS_TLS=true` when the provider requires TLS

Redis backs rate-limit counters and in-flight idempotency locks. Postgres remains the durable idempotency record store.

## Optional Tracing

- `OTEL_ENABLED=true`
- `OTEL_SERVICE_NAME=clearflow-api`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` or `http/protobuf`
- `OTEL_EXPORTER_OTLP_HEADERS=key=value,another=value`
- `OTEL_SAMPLE_RATIO=1.0`
- `OTEL_ENVIRONMENT=production`

Tracing is disabled by default. Local development and tests do not require a collector; when tracing is enabled without an endpoint, spans are created locally and discarded. Production fails fast if `OTEL_ENABLED=true` and the OTLP endpoint, service name, or protocol is invalid.

The API and worker propagate W3C trace context through HTTP requests, webhook handling, queued job payloads, and financial intelligence operations. Request and worker logs include `trace_id` and `span_id`.

## Local Full-Stack Startup

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

Open `http://localhost:3000`, click **Try Demo**, and use the dashboard, reconciliation, cash-flow, integrations, and ops pages.

## Local API Smoke Commands

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/demo-token | jq -r .token)
ORG_ID=$(curl -s http://localhost:8080/organizations -H "Authorization: Bearer $TOKEN" | jq -r '.[0].id')

curl -s -X POST http://localhost:8080/sync/stripe \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: stripe-demo-1" | jq

curl -s -X POST http://localhost:8080/sync/bank \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: bank-demo-1" | jq

curl -s -X POST http://localhost:8080/reconciliation/runs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: recon-demo-1" | jq

curl -s "http://localhost:8080/api/v1/jobs?organizationId=$ORG_ID" \
  -H "Authorization: Bearer $TOKEN" | jq

curl -s "http://localhost:8080/api/v1/audit-logs?organizationId=$ORG_ID" \
  -H "Authorization: Bearer $TOKEN" | jq

curl -s http://localhost:8080/api/v1/ops/metrics \
  -H "Authorization: Bearer $TOKEN" | jq
```

## Webhook Local Testing

Stripe mock webhook:

```bash
curl -s -X POST "http://localhost:8080/api/v1/webhooks/processors/stripe?organizationId=$ORG_ID" \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: t=1,v1=mock" \
  -d '{"id":"evt_demo_1","type":"balance.available"}' | jq
```

Plaid mock webhook in development:

```bash
curl -s -X POST http://localhost:8080/api/v1/webhooks/plaid \
  -H "Content-Type: application/json" \
  -H "X-Plaid-Mock-Webhook: true" \
  -d '{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE","item_id":"item_test","environment":"sandbox"}' | jq
```

## Tests

Backend:

```bash
cd services/api
gofmt -w .
go test ./...
go vet ./...
govulncheck ./...
```

Frontend:

```bash
cd apps/web
npm run lint
npm run typecheck
npm run build
npm run test
npm audit --audit-level=high
```

Full `npm audit` currently reports a moderate advisory in Next's internal PostCSS dependency. Do not run `npm audit fix --force`; npm suggests a breaking downgrade path. Track the Next patch release and keep the high-severity audit gate in CI.

## Production Startup

1. Apply migrations through `004_phase7_integrations.sql`.
2. Start the API process.
3. Start the worker process separately with `make worker`.
4. Configure Stripe webhook delivery to `POST /api/v1/webhooks/processors/stripe`.
5. Configure Plaid webhook delivery to `POST /api/v1/webhooks/plaid`.

## Operations

- Use `/api/v1/ops/metrics` for request, webhook, idempotency, job, and trace counters.
- Use `/api/v1/jobs`, `/api/v1/jobs/dead`, `/retry`, and `/cancel` for job operations.
- Use `/api/v1/audit-logs` for financial workflow audit history.
- Rotate provider credentials by disconnecting/reconnecting the provider integration.

## Common Failure Modes

- Frontend cannot reach API: if `NEXT_PUBLIC_API_BASE_URL` is configured, the UI shows clear error states; if it is omitted, Vercel demo mode uses sample data.
- Invalid CORS: set `ALLOWED_ORIGINS` to the deployed frontend origin in production.
- Missing `JWT_SECRET`: production startup fails.
- Missing `DATABASE_URL`: production startup fails.
- Redis enabled but unavailable: production startup fails; local dev falls back to memory.
- OTLP exporter misconfigured in production: API/worker startup fails before accepting work.
- Invalid Stripe signature: webhook returns `400` and increments Stripe failure metrics.
- Plaid verification required but missing mock/real signature: webhook returns `401`.
- Plaid/Stripe real mode vs mock mode: local mock flows work without real provider credentials; real provider calls require configured secrets and provider dashboard setup.
- Worker not processing jobs: confirm `make worker` is running, Postgres is reachable, and jobs are not scheduled for a future `run_after`.
- Reused OAuth state: callback returns `401`.
- Same idempotency key with different request body: returns `409`.
- Provider token encryption missing in production: startup/config validation fails.

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
