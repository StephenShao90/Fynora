# Debugging Clearflow

Clearflow now emits structured JSON logs from both the backend and browser.

## Backend Logs

Run the API from the repo root:

```bash
make api
```

When testing, copy the terminal lines that include:

- `http.request`
- `clearflow.operation`
- `reconciliation.engine.completed`
- `panic`
- `PLAID_ERROR`

Each request includes:

- `request_id`
- `method`
- `path`
- `status`
- `latency_ms`
- `user_id`

Reconciliation operations also include:

- `organization_id`
- `run_id`
- `matched_count`
- `exception_count`
- `evaluated_payouts`
- `evaluated_deposit_candidates`

## Browser Logs

Open DevTools Console. Frontend API calls log:

- `[clearflow-api]`
- `[clearflow-api:error]`

Send the object printed there, especially `path`, `status`, `durationMs`, and `requestId`.

## Protected Debug Snapshot

After logging in, you can inspect local state with:

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/debug/clearflow
```

This returns counts of payments, payouts, bank transactions, runs, matches, exceptions, and audit events. It does not include Plaid secrets or access tokens.
