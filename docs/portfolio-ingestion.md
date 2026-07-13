# Portfolio Ingestion

Clearflow supports manual holdings and CSV import first. This keeps the product safe for a portfolio demo because users never enter brokerage passwords and the app never scrapes Wealthsimple or any brokerage. When Postgres is configured, portfolio accounts, imports, holdings, and portfolio transactions persist in the database and survive API restarts.

Supported holdings columns include `account`, `account_type`, `symbol`, `ticker`, `name`, `security`, `security_type`, `asset_class`, `quantity`, `shares`, `average_cost`, `cost_basis`, `market_price`, `current_price`, `market_value`, `current_value`, and `currency`.

Supported portfolio transaction columns include `date`, `trade_date`, `transaction_date`, `account`, `symbol`, `ticker`, `action`, `activity`, `transaction_type`, `quantity`, `shares`, `price`, `trade_price`, `amount`, `net_amount`, `fees`, `commission`, `currency`, and `description`.

Portfolio review endpoints:

- `POST /portfolio/import/holdings-csv`
- `POST /portfolio/import/transactions-csv`
- `GET /portfolio/imports`
- `GET /portfolio/holdings`
- `GET /portfolio/transactions`
- `GET /portfolio/summary`
- `GET /portfolio/allocation`
- `GET /portfolio/risk`

The web Portfolio page now supports direct holdings and activity uploads, shows recent import row counts, and refreshes holdings, activity, allocation, and risk after each import.

Operational logs to check during manual testing:

- `portfolio.holdings_imported`
- `portfolio.transactions_imported`
- `GET /portfolio/holdings` returning `200`
- `GET /portfolio/transactions` returning `200`
- `GET /portfolio/imports` returning `200`

Future live integrations should use official APIs or approved aggregators such as Plaid Investments, SnapTrade, or Flinks. The backend includes a `BrokerageConnector` interface with manual, CSV, mock Plaid, and mock Wealthsimple CSV connectors.
