CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
    CREATE TYPE user_role AS ENUM ('member', 'admin');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT NOT NULL,
  email TEXT,
  password_hash TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'member',
  display_name TEXT,
  avatar_url TEXT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  data_root TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at TIMESTAMPTZ,
  CONSTRAINT users_email_unique UNIQUE (email),
  CONSTRAINT users_username_unique UNIQUE (username)
);

CREATE TABLE IF NOT EXISTS containers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  container_id TEXT NOT NULL,
  container_name TEXT NOT NULL,
  image TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'created',
  namespace TEXT NOT NULL DEFAULT 'default',
  auto_start BOOLEAN NOT NULL DEFAULT true,
  host_path TEXT,
  container_path TEXT NOT NULL DEFAULT '/data',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_started_at TIMESTAMPTZ,
  last_stopped_at TIMESTAMPTZ,
  CONSTRAINT containers_container_id_unique UNIQUE (container_id),
  CONSTRAINT containers_container_name_unique UNIQUE (container_name)
);

CREATE INDEX IF NOT EXISTS idx_containers_user_id ON containers(user_id);

CREATE TABLE IF NOT EXISTS snapshots (
  id TEXT PRIMARY KEY,
  container_id TEXT NOT NULL REFERENCES containers(container_id) ON DELETE CASCADE,
  parent_snapshot_id TEXT REFERENCES snapshots(id) ON DELETE SET NULL,
  snapshotter TEXT NOT NULL,
  digest TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS models (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  model_id TEXT NOT NULL,
  name TEXT,
  base_url TEXT NOT NULL,
  api_key TEXT NOT NULL,
  client_type TEXT NOT NULL,
  dimensions INTEGER,
  type TEXT NOT NULL DEFAULT 'chat',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT models_model_id_unique UNIQUE (model_id),
  CONSTRAINT models_type_check CHECK (type IN ('chat', 'embedding')),
  CONSTRAINT models_client_type_check CHECK (client_type IN ('openai', 'anthropic', 'google')),
  CONSTRAINT models_dimensions_check CHECK (type != 'embedding' OR dimensions IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_snapshots_container_id ON snapshots(container_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_parent_id ON snapshots(parent_snapshot_id);

CREATE TABLE IF NOT EXISTS container_versions (
  id TEXT PRIMARY KEY,
  container_id TEXT NOT NULL REFERENCES containers(container_id) ON DELETE CASCADE,
  snapshot_id TEXT NOT NULL REFERENCES snapshots(id) ON DELETE RESTRICT,
  version INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (container_id, version)
);

CREATE INDEX IF NOT EXISTS idx_container_versions_container_id ON container_versions(container_id);

CREATE TABLE IF NOT EXISTS lifecycle_events (
  id TEXT PRIMARY KEY,
  container_id TEXT NOT NULL REFERENCES containers(container_id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_events_container_id ON lifecycle_events(container_id);
CREATE INDEX IF NOT EXISTS idx_lifecycle_events_event_type ON lifecycle_events(event_type);
