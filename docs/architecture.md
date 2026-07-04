# Architecture

Fynora is organized as a monorepo with `apps/web` for the Next.js UI and `services/api` for the Go API.

```text
User -> Next.js Dashboard -> Go API
                              |-> PostgreSQL schema
                              |-> Raw Event Store / S3
                              |-> Spending Intelligence Engine
                              |-> Portfolio Analytics Engine
                              |-> Market Data Provider
                              |-> Optional AI Advisor API
```

The backend separates domain engines (`advisor`, `portfolio`, `marketdata`, `storage`, `auth`) from HTTP wiring. The current MVP uses a local in-process demo store so reviewers can run the app instantly; migrations define the production PostgreSQL schema.
