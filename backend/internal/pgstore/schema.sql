-- Idempotent schema, applied at every server startup (docs/adr/0001).
-- Embedding columns are vector(384): bge-small-en-v1.5 (docs/adr/0002).

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS users (
    id                  TEXT PRIMARY KEY,
    username            TEXT,
    email               TEXT NOT NULL UNIQUE,
    password_hash       TEXT NOT NULL,
    name                TEXT NOT NULL DEFAULT '',
    roles               TEXT[] NOT NULL DEFAULT '{}',
    granted_permissions TEXT[] NOT NULL DEFAULT '{}',
    denied_permissions  TEXT[] NOT NULL DEFAULT '{}',
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by          TEXT NOT NULL DEFAULT '',
    last_login          TIMESTAMPTZ
);

-- username was added after the initial release; backfill any pre-existing
-- rows (from an older schema version) before enforcing NOT NULL + unique.
ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT;
UPDATE users SET username = split_part(email, '@', 1) WHERE username IS NULL;
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_username_key ON users (username);

-- Signup-approval workflow: new self-signups start 'pending' until a
-- superuser approves them (requested_role holds what they asked for, since
-- `roles` stays empty until approved — see auth.GenerateToken). Default
-- 'approved' so admin-created accounts (a superuser already made that call)
-- and existing rows from before this column existed need no extra step;
-- retroactively marking pre-existing accounts pending is a separate,
-- one-time, manually-run step (cmd/backfill-approval), not part of this
-- idempotent schema.
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'approved';
ALTER TABLE users ADD COLUMN IF NOT EXISTS requested_role TEXT NOT NULL DEFAULT '';

-- Sequential, zero-padded signup IDs ("00", "01", "02", ...). Seeded/admin-
-- created accounts (e.g. "superuser-001") use their own non-numeric IDs and
-- never touch this sequence.
CREATE SEQUENCE IF NOT EXISTS user_id_seq START WITH 1;

CREATE TABLE IF NOT EXISTS roles (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    permissions TEXT[] NOT NULL DEFAULT '{}'
);

-- Corti custom templates: key/name lifted out for listing, the full config
-- kept as one JSONB document (the app only ever reads/writes it whole).
CREATE TABLE IF NOT EXISTS corti_templates (
    key        TEXT PRIMARY KEY,
    name       TEXT NOT NULL DEFAULT '',
    definition JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type           TEXT NOT NULL,
    title          TEXT NOT NULL,
    transcript     TEXT NOT NULL DEFAULT '',
    document       JSONB,
    extraction     JSONB,
    interaction_id TEXT NOT NULL DEFAULT '',
    -- Embeddings are computed asynchronously; NULL means pending.
    -- transcript_embedding is doc-level (spec compliance / future "related
    -- sessions"); search itself queries chunks + extraction_embedding.
    transcript_embedding vector(384),
    -- extraction_text is the exact serialization extraction_embedding was
    -- computed from, kept so the vector's source text is always inspectable.
    extraction_text      TEXT,
    extraction_embedding vector(384),
    embedded_at          TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);
CREATE INDEX IF NOT EXISTS sessions_extraction_embedding_idx
    ON sessions USING hnsw (extraction_embedding vector_cosine_ops);

CREATE TABLE IF NOT EXISTS transcript_chunks (
    id              BIGSERIAL PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    chunk_index     INT NOT NULL,
    chunk_text      TEXT NOT NULL,
    chunk_embedding vector(384) NOT NULL,
    UNIQUE (session_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS transcript_chunks_session_idx ON transcript_chunks (session_id);
CREATE INDEX IF NOT EXISTS transcript_chunks_embedding_idx
    ON transcript_chunks USING hnsw (chunk_embedding vector_cosine_ops);

-- Lifetime per-feature usage counters for the demo ("user") role — 3 free
-- tries per feature, enforced by middleware.RequireDemoAllowance. Doctors,
-- admins, and superusers are never subject to this and never get rows here.
CREATE TABLE IF NOT EXISTS demo_usage (
    user_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature   TEXT NOT NULL,
    use_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, feature)
);
