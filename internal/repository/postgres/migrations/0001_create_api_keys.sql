-- api_keys: API key store (only SHA-256 hashes are persisted)
CREATE TABLE IF NOT EXISTS api_keys (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    key_hash      TEXT NOT NULL UNIQUE,
    scopes        TEXT[] NOT NULL DEFAULT '{}',
    expires_at    TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    request_count BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_created_at ON api_keys (created_at);
