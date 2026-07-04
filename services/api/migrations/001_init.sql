CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS advisor_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  country TEXT NOT NULL CHECK (country IN ('US', 'CA')),
  age INT,
  monthly_income_estimate NUMERIC,
  risk_tolerance TEXT CHECK (risk_tolerance IN ('conservative', 'moderate', 'aggressive')),
  emergency_fund_months_target INT DEFAULT 6,
  current_emergency_fund NUMERIC DEFAULT 0,
  has_high_interest_debt BOOLEAN DEFAULT false,
  high_interest_debt_amount NUMERIC DEFAULT 0,
  high_interest_debt_apr NUMERIC DEFAULT 0,
  has_employer_match BOOLEAN DEFAULT false,
  employer_match_description TEXT,
  retirement_account_access TEXT,
  primary_goal TEXT,
  investment_time_horizon_years INT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS raw_imports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  import_type TEXT NOT NULL CHECK (import_type IN ('transactions', 'holdings', 'portfolio_transactions')),
  original_filename TEXT,
  raw_storage_key TEXT,
  row_count INT NOT NULL DEFAULT 0,
  imported_count INT NOT NULL DEFAULT 0,
  failed_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  account_id TEXT,
  amount NUMERIC NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('income', 'expense')),
  currency TEXT NOT NULL,
  merchant TEXT,
  normalized_merchant TEXT,
  category TEXT,
  description TEXT,
  occurred_at TIMESTAMP NOT NULL,
  raw_event_key TEXT,
  import_id UUID REFERENCES raw_imports(id) ON DELETE SET NULL,
  metadata JSONB DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS brokerage_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider TEXT NOT NULL CHECK (provider IN ('manual', 'wealthsimple_csv', 'plaid', 'snaptrade', 'flinks', 'other')),
  account_name TEXT NOT NULL,
  account_type TEXT NOT NULL CHECK (account_type IN ('TFSA', 'RRSP', 'FHSA', 'taxable', 'cash', 'margin', '401k', 'IRA', 'Roth IRA', 'other')),
  currency TEXT NOT NULL,
  institution_name TEXT,
  connection_status TEXT NOT NULL CHECK (connection_status IN ('manual', 'imported', 'connected', 'disconnected')),
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS holdings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  brokerage_account_id UUID REFERENCES brokerage_accounts(id) ON DELETE CASCADE,
  symbol TEXT NOT NULL,
  security_name TEXT,
  security_type TEXT NOT NULL CHECK (security_type IN ('stock', 'etf', 'mutual_fund', 'crypto', 'cash', 'other')),
  quantity NUMERIC NOT NULL,
  average_cost NUMERIC,
  currency TEXT NOT NULL,
  market_value NUMERIC,
  last_price NUMERIC,
  price_as_of TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS portfolio_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  brokerage_account_id UUID REFERENCES brokerage_accounts(id) ON DELETE CASCADE,
  symbol TEXT,
  transaction_type TEXT NOT NULL CHECK (transaction_type IN ('buy', 'sell', 'dividend', 'deposit', 'withdrawal', 'fee', 'transfer', 'split', 'other')),
  quantity NUMERIC,
  price NUMERIC,
  amount NUMERIC,
  fees NUMERIC,
  currency TEXT,
  occurred_at TIMESTAMP NOT NULL,
  description TEXT,
  import_id UUID REFERENCES raw_imports(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS portfolio_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  total_market_value NUMERIC,
  cash_value NUMERIC,
  invested_value NUMERIC,
  total_cost_basis NUMERIC,
  unrealized_gain_loss NUMERIC,
  unrealized_gain_loss_pct NUMERIC,
  asset_allocation JSONB,
  sector_allocation JSONB,
  currency_allocation JSONB,
  top_holdings JSONB,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  severity TEXT NOT NULL,
  metadata JSONB DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_occurred ON transactions(user_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_transactions_user_category ON transactions(user_id, category);
CREATE INDEX IF NOT EXISTS idx_transactions_user_merchant ON transactions(user_id, normalized_merchant);
CREATE INDEX IF NOT EXISTS idx_transactions_user_direction ON transactions(user_id, direction);
CREATE INDEX IF NOT EXISTS idx_holdings_user_symbol ON holdings(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_holdings_user_account ON holdings(user_id, brokerage_account_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_tx_user_occurred ON portfolio_transactions(user_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_portfolio_tx_user_symbol ON portfolio_transactions(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_brokerage_accounts_user ON brokerage_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_raw_imports_user_created ON raw_imports(user_id, created_at);
