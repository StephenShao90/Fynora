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

CREATE TABLE IF NOT EXISTS plaid_connections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id TEXT NOT NULL,
  institution_name TEXT NOT NULL,
  access_token_ciphertext TEXT NOT NULL,
  cursor TEXT,
  products TEXT[] NOT NULL DEFAULT ARRAY['transactions'],
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  last_synced_at TIMESTAMP
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
CREATE INDEX IF NOT EXISTS idx_plaid_connections_user ON plaid_connections(user_id);

CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  currency TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organization_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'analyst', 'viewer')),
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(organization_id, user_id)
);

CREATE TABLE IF NOT EXISTS processor_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  processor TEXT NOT NULL CHECK (processor IN ('stripe', 'square', 'paypal', 'manual')),
  external_account_id TEXT NOT NULL,
  display_name TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(organization_id, processor, external_account_id)
);

CREATE TABLE IF NOT EXISTS bank_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider TEXT NOT NULL CHECK (provider IN ('plaid', 'csv', 'manual')),
  external_account_id TEXT NOT NULL,
  institution_name TEXT,
  account_name TEXT,
  mask TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(organization_id, provider, external_account_id)
);

CREATE TABLE IF NOT EXISTS customers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT,
  email TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invoices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
  number TEXT,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  status TEXT NOT NULL,
  due_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  processor TEXT NOT NULL,
  processor_payment_id TEXT NOT NULL,
  customer_email TEXT,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  status TEXT NOT NULL,
  occurred_at TIMESTAMP NOT NULL,
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(organization_id, processor_payment_id)
);

CREATE TABLE IF NOT EXISTS refunds (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  processor_refund_id TEXT NOT NULL,
  payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  occurred_at TIMESTAMP NOT NULL,
  UNIQUE(organization_id, processor_refund_id)
);

CREATE TABLE IF NOT EXISTS fees (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  processor_fee_id TEXT NOT NULL,
  payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  occurred_at TIMESTAMP NOT NULL,
  description TEXT,
  UNIQUE(organization_id, processor_fee_id)
);

CREATE TABLE IF NOT EXISTS payouts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  processor TEXT NOT NULL,
  processor_payout_id TEXT NOT NULL,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  status TEXT NOT NULL,
  expected_arrival_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(organization_id, processor_payout_id)
);

CREATE TABLE IF NOT EXISTS payout_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  payout_id UUID NOT NULL REFERENCES payouts(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL CHECK (source_type IN ('payment', 'refund', 'fee', 'adjustment')),
  source_id TEXT NOT NULL,
  amount_minor BIGINT NOT NULL,
  currency TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bank_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  source TEXT NOT NULL,
  external_id TEXT NOT NULL,
  amount_minor BIGINT NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('credit', 'debit')),
  currency TEXT NOT NULL,
  description TEXT,
  posted_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(organization_id, external_id)
);

CREATE TABLE IF NOT EXISTS reconciliation_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  matched_count INT NOT NULL DEFAULT 0,
  exception_count INT NOT NULL DEFAULT 0,
  started_at TIMESTAMP NOT NULL DEFAULT now(),
  completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS reconciliation_matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  run_id UUID NOT NULL REFERENCES reconciliation_runs(id) ON DELETE CASCADE,
  payout_id UUID REFERENCES payouts(id) ON DELETE SET NULL,
  bank_transaction_id UUID REFERENCES bank_transactions(id) ON DELETE SET NULL,
  amount_minor BIGINT NOT NULL,
  confidence NUMERIC NOT NULL,
  explanation TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reconciliation_exceptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  run_id UUID REFERENCES reconciliation_runs(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  severity TEXT NOT NULL,
  title TEXT NOT NULL,
  explanation TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  reference_id TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exception_notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  exception_id UUID NOT NULL REFERENCES reconciliation_exceptions(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  key TEXT NOT NULL,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  request_hash TEXT NOT NULL,
  status_code INT NOT NULL,
  response_body TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  PRIMARY KEY(user_id, key)
);

CREATE TABLE IF NOT EXISTS webhook_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider TEXT NOT NULL CHECK (provider IN ('stripe', 'plaid')),
  external_event_id TEXT NOT NULL,
  organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  processed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(provider, external_event_id)
);

CREATE TABLE IF NOT EXISTS sync_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  source TEXT NOT NULL CHECK (source IN ('stripe', 'plaid', 'csv', 'manual')),
  status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed')),
  attempts INT NOT NULL DEFAULT 0,
  error TEXT,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  target_type TEXT,
  target_id TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orgs_user ON organizations(user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_processor_accounts_org ON processor_accounts(organization_id);
CREATE INDEX IF NOT EXISTS idx_bank_accounts_org ON bank_accounts(organization_id);
CREATE INDEX IF NOT EXISTS idx_payments_org_occurred ON payments(organization_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_payouts_org_arrival ON payouts(organization_id, expected_arrival_at);
CREATE INDEX IF NOT EXISTS idx_payout_items_payout ON payout_items(payout_id);
CREATE INDEX IF NOT EXISTS idx_bank_tx_org_posted ON bank_transactions(organization_id, posted_at);
CREATE INDEX IF NOT EXISTS idx_recon_runs_org_started ON reconciliation_runs(organization_id, started_at);
CREATE INDEX IF NOT EXISTS idx_recon_exceptions_org_status ON reconciliation_exceptions(organization_id, status);
CREATE INDEX IF NOT EXISTS idx_webhook_events_processed ON webhook_events(provider, processed_at);
CREATE INDEX IF NOT EXISTS idx_sync_jobs_status ON sync_jobs(status, created_at);
