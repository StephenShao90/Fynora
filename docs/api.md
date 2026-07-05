# API

All protected endpoints require `Authorization: Bearer <token>`.

Health: `GET /health`

Auth: `POST /auth/register`, `POST /auth/login`, `POST /auth/demo-token`

User: `GET /me`, `GET /me/advisor-profile`, `PUT /me/advisor-profile`

Transactions: `POST /imports/transactions-csv`, `GET /imports`, `GET /imports/{id}`, `POST /transactions`, `GET /transactions`, `GET /transactions/{id}`, `PATCH /transactions/{id}/category`, `DELETE /transactions/{id}`

Insights: `GET /insights/monthly-summary`, `GET /insights/categories`, `GET /insights/merchants`, `GET /insights/subscriptions`, `GET /insights/anomalies`, `GET /insights/duplicate-charges`, `GET /insights/cash-flow`

Advisor: `GET /advisor/plan`, `GET /advisor/emergency-fund`, `GET /advisor/account-priority`, `POST /advisor/investment-projection`, `POST /advisor/chat`, `POST /advisor/monthly-summary`

Portfolio: `POST /portfolio/accounts`, `GET /portfolio/accounts`, `POST /portfolio/import/holdings-csv`, `POST /portfolio/import/transactions-csv`, `GET /portfolio/holdings`, `GET /portfolio/summary`, `GET /portfolio/allocation`, `GET /portfolio/performance`, `GET /portfolio/risk`, `GET /portfolio/rebalance-suggestions`, `GET /portfolio/projected-growth`

Market: `GET /market/quote/{symbol}`, `POST /market/quotes`

Connections: `GET /connections`, `DELETE /connections/{id}`, `POST /connections/plaid/link-token`, `POST /connections/plaid/exchange-public-token`, `POST /connections/plaid/sync-transactions`
