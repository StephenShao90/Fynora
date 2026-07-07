CREATE TABLE IF NOT EXISTS plaid_webhook_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
  plaid_item_id UUID REFERENCES plaid_items(id) ON DELETE SET NULL,
  webhook_type TEXT NOT NULL,
  webhook_code TEXT NOT NULL,
  item_id TEXT,
  environment TEXT,
  payload_json JSONB NOT NULL,
  dedupe_key TEXT UNIQUE NOT NULL,
  received_at TIMESTAMP NOT NULL DEFAULT now(),
  processed_at TIMESTAMP,
  status TEXT NOT NULL DEFAULT 'received',
  error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_plaid_webhook_events_created ON plaid_webhook_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_plaid_webhook_events_item ON plaid_webhook_events(item_id);

CREATE TABLE IF NOT EXISTS outbox_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
  event_type TEXT NOT NULL,
  aggregate_type TEXT,
  aggregate_id TEXT,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'published', 'failed', 'dead')),
  attempts INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 3,
  available_at TIMESTAMP NOT NULL DEFAULT now(),
  published_at TIMESTAMP,
  error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending ON outbox_events(status, available_at, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_events_org ON outbox_events(organization_id, created_at DESC);

CREATE TABLE IF NOT EXISTS processor_webhook_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
  provider TEXT NOT NULL,
  event_type TEXT NOT NULL,
  external_event_id TEXT,
  payload_json JSONB NOT NULL,
  dedupe_key TEXT UNIQUE NOT NULL,
  status TEXT NOT NULL DEFAULT 'received',
  received_at TIMESTAMP NOT NULL DEFAULT now(),
  processed_at TIMESTAMP,
  error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processor_webhook_events_provider ON processor_webhook_events(provider, created_at DESC);

ALTER TABLE sync_jobs DROP CONSTRAINT IF EXISTS sync_jobs_status_check;
ALTER TABLE sync_jobs ADD CONSTRAINT sync_jobs_status_check CHECK (status IN ('queued', 'running', 'completed', 'failed', 'dead', 'cancelled', 'cancel_requested'));
