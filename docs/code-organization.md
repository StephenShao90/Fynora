# Code Organization

Clearflow is organized as a monorepo with a Next.js operations UI and a Go API/worker backend.

## Frontend

`apps/web/app`

Route-level pages only. Keep page files focused on loading page data, composing sections, and handling route-specific actions.

`apps/web/components/layout`

Shared application chrome and layout primitives:

- `Shell`
- `Card`
- `Metric`
- `Header`
- `Empty`
- `ToastProvider`

`apps/web/components/help`

Contextual help behavior:

- `HelpFlow` toggles page guide mode.
- `GuideMarker` renders numbered contextual guide markers only while guide mode is active.

`apps/web/components/demo`

Demo-specific workflow helpers such as `DemoPilot`.

`apps/web/components/insights`, `payouts`, `reconciliation`

Feature-specific reusable panels. Put components here when they are reusable across pages but still tied to a product domain.

`apps/web/lib`

Client-side API access, demo fallback behavior, and provider helper functions. Keep UI state out of this directory.

## Backend

`services/api/cmd/api`

HTTP server composition and route handlers. Route files are grouped by product area where practical.

`services/api/cmd/worker`

Background worker entrypoint for async sync/reconciliation jobs.

`services/api/internal`

Backend domain packages. Prefer putting reusable business logic here instead of inside route handlers:

- `reconciliation`
- `cashflow`
- `insights`
- `recommendations`
- `portfolio`
- `processors`
- `plaid`
- `repository`

`services/api/migrations`

Ordered PostgreSQL migrations. Always update `Makefile migrate` when adding a migration.

## Rule Of Thumb

Pages compose. Components display. `lib` talks to APIs. Backend handlers translate HTTP into domain/repository calls. Domain packages contain the logic worth testing directly.
