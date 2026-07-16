# Performance Model

Clearflow optimizes the local product demo for fast first paint, predictable API behavior, and low-stale-state risk after write actions.

## Frontend Data Loading

- Dashboard metrics load through `GET /api/v1/dashboard/summary` instead of several independent startup requests.
- Demo fallback mode returns seeded demo responses immediately when no `NEXT_PUBLIC_API_BASE_URL` is configured.
- `useApi` deduplicates in-flight reads and keeps successful responses in a short-lived client cache.
- Successful non-GET requests and file uploads invalidate the shared client cache so follow-up screens do not show stale operational state.

## Request Timeouts

- Local demo reads use a short timeout so the UI can fall back quickly when the backend is not running.
- Configured API environments use a longer read timeout suitable for deployed services.
- Write requests use a longer timeout because sync, reconciliation, and upload workflows can legitimately take more time.

## Rendering

- Heavy chart modules are dynamically imported on insight-heavy pages.
- The shell prefetches primary product routes after hydration.
- Page transitions use lightweight CSS animation only; no full-page reload is required for normal app navigation or onboarding updates.

## Operational Expectations

The product should feel usable even while local infrastructure is offline, but production correctness still comes from the API, Postgres, Redis-backed workers, and provider integrations. The frontend cache is intentionally small and ephemeral; server state remains the source of truth.
