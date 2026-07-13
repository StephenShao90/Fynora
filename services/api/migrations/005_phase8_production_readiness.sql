CREATE TABLE IF NOT EXISTS import_errors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  import_id UUID NOT NULL REFERENCES raw_imports(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  row_number INT NOT NULL,
  field TEXT,
  code TEXT NOT NULL,
  message TEXT NOT NULL,
  raw_row JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_import_errors_import ON import_errors(import_id, row_number);
CREATE INDEX IF NOT EXISTS idx_import_errors_user ON import_errors(user_id, created_at);
