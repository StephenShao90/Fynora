CREATE TABLE IF NOT EXISTS organization_setup (
  organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
  selected_scenario TEXT,
  business_type TEXT,
  checklist JSONB NOT NULL DEFAULT '{}'::jsonb,
  completed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_exception_notes_exception_created ON exception_notes(exception_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_exception_notes_org_created ON exception_notes(organization_id, created_at DESC);
