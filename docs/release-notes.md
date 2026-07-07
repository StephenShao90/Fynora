# Release Notes

## Phase 1: API Foundation

Created the initial Go API, core financial data model, local demo flows, and Next.js dashboard foundation.

## Phase 2: Auth, Organizations, and RBAC

Added users, authentication, organizations, membership roles, and organization-scoped access patterns.

## Phase 3: Sessions, Rate Limits, Idempotency, and Jobs

Added refresh sessions, idempotent financial write behavior, async sync/reconciliation jobs, audit logs, and job inspection endpoints.

## Phase 4: Webhooks, Ops, and Security

Added Plaid/processor webhook handling, deduplication, operational metrics, job retry/cancel/dead-letter surfaces, and production config validation.

## Phase 5: Financial Intelligence API

Added payout explanation, reconciliation scoring, anomaly detection, spending insights, cash recommendations, and cash-flow forecasting endpoints.

## Phase 6: Frontend Intelligence UI

Added frontend pages and components for payout explanations, reconciliation match reasoning, forecasts, anomalies, recommendations, and spending insights.

## Phase 7: Real Integration Hardening

Added Stripe OAuth-style connection flow, protected provider tokens, webhook verification, Plaid webhook verification controls, and integration status APIs.

## Phase 8: Redis, OTEL Readiness, Stripe/Plaid Polish, and Frontend Tests

Added optional Redis-backed rate limiting/idempotency locks, tracing readiness, integration UI, and Vitest/Testing Library coverage.

## Phase 9: OTLP Export, Dependency Security, and CI Hardening

Added real optional OTLP trace export, trace propagation through jobs/webhooks/financial operations, graceful API/worker tracer shutdown, dependency vulnerability cleanup, Docker build artifacts, and stronger CI gates.

## Phase 10: Deployable Demo, Repo Cleanup, and Interview Walkthrough

Polished Vercel demo behavior, added sample operational data, added Ops UI, documented deployment modes, expanded runbook/demo guidance, added release notes, and prepared the accumulated work for a final productionization commit.
