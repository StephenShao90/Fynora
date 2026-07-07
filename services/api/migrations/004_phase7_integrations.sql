CREATE TABLE IF NOT EXISTS provider_connections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  external_account_id TEXT NOT NULL,
  display_name TEXT,
  status TEXT NOT NULL DEFAULT 'connected' CHECK (status IN ('connected', 'disconnected', 'error')),
  access_token_ciphertext TEXT,
  refresh_token_ciphertext TEXT,
  scopes TEXT,
  connected_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  connected_at TIMESTAMP,
  disconnected_at TIMESTAMP,
  last_sync_at TIMESTAMP,
  last_error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE (organization_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_provider_connections_org ON provider_connections(organization_id, provider);

CREATE TABLE IF NOT EXISTS oauth_states (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  state_hash TEXT NOT NULL UNIQUE,
  redirect_uri TEXT NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  used_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_provider_hash ON oauth_states(provider, state_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expiry ON oauth_states(expires_at);
