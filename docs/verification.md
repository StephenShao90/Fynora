# Verification Flow

Use this when you want to prove Clearflow works end to end and send logs back for debugging.

## 1. Start The Local Stack

Terminal 1:

```bash
make local-env
make docker-up
make migrate
make api
```

Expected API logs:

```text
database.connected
Clearflow API listening on :8080
http.request
```

Terminal 2:

```bash
make worker
```

Expected worker logs:

```text
worker.started
worker.job.started
worker.job.completed
```

Worker job logs only appear when queued jobs exist.

Terminal 3:

```bash
make web
```

Expected web logs:

```text
Next.js 16
Local: http://localhost:3000
```

## 2. Run Automated Smoke Verification

Terminal 4:

```bash
make smoke
```

The smoke runner prints one line per feature:

```text
[SMOKE] PASS health endpoint responds
[SMOKE] PASS demo auth creates token
[SMOKE] PASS organization is available
[SMOKE] PASS stripe mock sync imports processor data
[SMOKE] PASS bank mock sync imports bank data
[SMOKE] PASS reconciliation run works
[SMOKE] PASS payout ledger loads
[SMOKE] PASS payout explanation loads
[SMOKE] PASS cash-flow forecast loads
[SMOKE] PASS anomalies load
[SMOKE] PASS cash recommendations load
[SMOKE] PASS reconciliation match scoring loads
[SMOKE] PASS jobs list loads
[SMOKE] PASS async worker jobs complete
[SMOKE] PASS audit logs load
[SMOKE] PASS ops metrics load
[SMOKE] PASS idempotency replay returns same stripe sync result
```

At the end it prints a `requestIdsPrefix`, such as:

```text
smoke-20260711201030-
```

Use that prefix to find matching API logs.

## 3. Logs To Send Back

If anything fails, send:

- the full `make smoke` output
- API terminal lines containing `http.request`
- API terminal lines containing the smoke request ID prefix
- worker terminal lines containing `worker.job`
- the `asyncJobIds` from the final `LOOK_FOR_IN_WORKER_TERMINAL` smoke output
- any frontend browser console lines starting with `[clearflow-api]` or `[clearflow-api:error]`

Useful terminal filters:

```bash
# API terminal output is easiest to copy directly, but if saved to a file:
grep 'smoke-' api.log
grep 'http.request' api.log
grep 'database.connected' api.log

# Worker log file if saved:
grep 'worker.job' worker.log
```

## 4. Manual Browser Verification

After `make web`, open `http://localhost:3000`.

1. Click **Try Demo**.
2. Dashboard should load cash, payout, forecast, and exception widgets.
3. Open **Reconciliation**.
4. Click **Sync processor**.
5. Click **Sync bank**.
6. Click **Reconcile**.
7. Click **View explanation** on a payout.
8. Open **Cash Flow** and check forecast, anomalies, recommendations, spending insights.
9. Open **Ops** and check jobs, audit logs, metrics.
10. Open **Integrations** and check Stripe/Plaid status cards.

Frontend browser console logs:

```text
[clearflow-api] { path, method, status, durationMs, requestId }
[clearflow-api:error] { path, status, durationMs, requestId, message }
[clearflow-api:demo-fallback] { path, method, requestId, reason }
```

`[clearflow-api:error]` is a warning, not a Next.js crash overlay. If it appears, copy the object and the matching API `request_id` log.

## 5. Unit And Build Tests

Full local quality gate:

```bash
make verify
```

Backend-only fallback:

```bash
make verify-backend
```

Frontend-only fallback:

```bash
make verify-frontend
```

Known audit note: full `npm audit` can still report a moderate Next/PostCSS advisory. High-severity audit should pass.

## 6. Render Verification

For Render API:

```bash
API_BASE=https://your-api.onrender.com node scripts/smoke-clearflow.mjs
```

If Render is using internal Postgres/Redis URLs, run the smoke script against the public API URL, not directly against internal database or Redis URLs.
