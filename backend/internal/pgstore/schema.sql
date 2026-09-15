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

-- Patient identity — added for NHS/SystmOne integration (docs/nhs/). NHS
-- number, DOB and name are stored as opaque ciphertext produced by
-- internal/cryptofield (AES-256-GCM, envelope key from FIELD_ENCRYPTION_KEY)
-- rather than plaintext, since this is the single most sensitive identifier
-- in the whole schema. nhs_number_hash is a deterministic HMAC-SHA256 of the
-- normalized NHS number, kept alongside the ciphertext purely so the app can
-- look a patient up / enforce uniqueness by NHS number without ever
-- decrypting every row to compare.
CREATE TABLE IF NOT EXISTS patients (
    id                TEXT PRIMARY KEY,
    nhs_number_enc    TEXT,           -- ciphertext; NULL until PDS-verified or manually entered
    nhs_number_hash   TEXT,           -- HMAC(nhs_number) for lookup/uniqueness, NULL if nhs_number_enc is NULL
    name_enc          TEXT NOT NULL,  -- ciphertext
    date_of_birth_enc TEXT,           -- ciphertext, NULL if unknown
    sex               TEXT NOT NULL DEFAULT 'unknown', -- FHIR AdministrativeGender: male/female/other/unknown
    pds_verified_at   TIMESTAMPTZ,    -- last time a PDS trace confirmed this identity, NULL if never verified
    created_by        TEXT NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS patients_nhs_number_hash_key ON patients (nhs_number_hash) WHERE nhs_number_hash IS NOT NULL;

-- Sessions may now optionally be linked to a structured patient record
-- (NHS-number-based) instead of, or in addition to, the pre-existing
-- free-text patientName carried in a generated document's own JSON — that
-- field is unaffected and still works for non-NHS document generation.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS patient_id TEXT REFERENCES patients(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS sessions_patient_id_idx ON sessions (patient_id);

-- Clinical audit trail — append-only by convention (no UPDATE/DELETE is
-- exposed anywhere in internal/audit or internal/pgstore/audit_store.go).
-- This is deliberately separate from internal/middleware/logger.go, which is
-- operational request logging, not a "who viewed/exported/sent which
-- patient's data, when" clinical record.
CREATE TABLE IF NOT EXISTS audit_log (
    id           BIGSERIAL PRIMARY KEY,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id     TEXT NOT NULL,             -- users.id; not a FK so a deleted user's history survives
    actor_name   TEXT NOT NULL DEFAULT '',  -- denormalized snapshot, same reasoning
    action       TEXT NOT NULL,             -- e.g. "session.view", "session.export", "patient.view", "nhs.send_document"
    resource_type TEXT NOT NULL,            -- e.g. "session", "patient", "document"
    resource_id  TEXT NOT NULL DEFAULT '',
    patient_id   TEXT,                      -- set when the action touched a specific patient's data
    ip_address   TEXT NOT NULL DEFAULT '',
    detail       JSONB
);

CREATE INDEX IF NOT EXISTS audit_log_occurred_at_idx ON audit_log (occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_actor_id_idx ON audit_log (actor_id);
CREATE INDEX IF NOT EXISTS audit_log_patient_id_idx ON audit_log (patient_id) WHERE patient_id IS NOT NULL;
