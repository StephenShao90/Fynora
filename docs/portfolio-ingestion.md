# Portfolio Ingestion

Fynora supports manual holdings and CSV import first. This keeps the product safe for a portfolio demo because users never enter brokerage passwords and the app never scrapes Wealthsimple or any brokerage.

Supported holdings columns include `account`, `account_type`, `symbol`, `name`, `security_type`, `quantity`, `average_cost`, `market_price`, `market_value`, and `currency`.

Supported portfolio transaction columns include `date`, `account`, `symbol`, `action`, `quantity`, `price`, `amount`, `fees`, `currency`, and `description`.

Future live integrations should use official APIs or approved aggregators such as Plaid Investments, SnapTrade, or Flinks. The backend includes a `BrokerageConnector` interface with manual, CSV, mock Plaid, and mock Wealthsimple CSV connectors.
