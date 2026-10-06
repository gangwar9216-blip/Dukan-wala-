ALTER TABLE partners ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_partners_owner ON partners(owner_user_id);

CREATE TABLE IF NOT EXISTS medical_help_events (
  id UUID PRIMARY KEY, request_id UUID NOT NULL REFERENCES medical_help_requests(id) ON DELETE CASCADE,
  actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL, status VARCHAR(30) NOT NULL, internal_note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_help_events_request_created ON medical_help_events(request_id, created_at DESC);

CREATE TABLE IF NOT EXISTS device_tokens (
  id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE, platform VARCHAR(20) NOT NULL DEFAULT 'ANDROID',
  active BOOLEAN NOT NULL DEFAULT TRUE, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_device_tokens_user_active ON device_tokens(user_id, active);
