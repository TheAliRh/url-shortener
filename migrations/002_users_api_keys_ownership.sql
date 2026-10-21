CREATE TABLE IF NOT EXISTS users (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash     BYTEA       NOT NULL UNIQUE,   -- SHA-256 of the key, never the key itself
    prefix       TEXT        NOT NULL,          -- first characters, for display only
    name         TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);

-- Existing rows need an owner: create a legacy user and assign them
INSERT INTO users (email) VALUES ('legacy@local') ON CONFLICT DO NOTHING;

ALTER TABLE urls ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
UPDATE urls SET user_id = (SELECT id FROM users WHERE email = 'legacy@local') WHERE user_id IS NULL;
ALTER TABLE urls ALTER COLUMN user_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS urls_user_id_id_idx ON urls (user_id, id DESC);