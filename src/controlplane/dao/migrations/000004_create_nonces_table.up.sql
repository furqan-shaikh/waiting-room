CREATE TABLE IF NOT EXISTS nonces(
    key_id TEXT NOT NULL,
    nonce_value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (key_id, nonce_value)
);