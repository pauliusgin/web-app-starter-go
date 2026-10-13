-- +goose Up
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  email_verified_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,
  signin_at TIMESTAMPTZ
);

CREATE TABLE event_log (
  id UUID PRIMARY KEY,
  user_id UUID,
  event_label TEXT NOT NULL,
  event_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  token_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

CREATE TABLE auth_challenges (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  target_type TEXT NOT NULL,
  target TEXT NOT NULL,
  purpose TEXT NOT NULL,
  token_hash TEXT,
  code TEXT,
  provider TEXT,
  reference_id TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  CONSTRAINT auth_challenges_has_credential CHECK (
    token_hash IS NOT NULL
    OR code IS NOT NULL
  )
);

CREATE INDEX idx_auth_challenges_user_id ON auth_challenges (user_id);

CREATE TABLE messaging_log (
  id UUID PRIMARY KEY,
  recipient_id UUID NOT NULL,
  recipient_email TEXT,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  type TEXT NOT NULL,
  method TEXT NOT NULL,
  status TEXT NOT NULL,
  errors JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  reference_id TEXT
);

CREATE TABLE uploads (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  purpose TEXT NOT NULL,
  kind TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  width INT,
  height INT,
  original_path TEXT NOT NULL,
  original_url TEXT NOT NULL,
  preview_path TEXT,
  preview_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  attached_at TIMESTAMPTZ
);

CREATE INDEX idx_uploads_user_id ON uploads (user_id);

CREATE INDEX idx_uploads_pending_created_at ON uploads (created_at)
WHERE
  attached_at IS NULL;

CREATE TABLE profiles (
  id UUID PRIMARY KEY REFERENCES users (id),
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE profiles;
DROP TABLE uploads;
DROP TABLE messaging_log;
DROP TABLE auth_challenges;
DROP TABLE refresh_tokens;
DROP TABLE event_log;
DROP TABLE users;
