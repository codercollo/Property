CREATE TABLE IF NOT EXISTS revoked_tokens (
    id bigserial PRIMARY KEY,
    token_hash bytea NOT NULL UNIQUE,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    revoked_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    expires_at timestamp(0) with time zone NOT NULL
);

-- Index for fast lookup when authenticating
CREATE INDEX idx_revoked_tokens_hash ON revoked_tokens(token_hash);

-- Index for cleanup of expired tokens
CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);

-- Index for user-specific queries
CREATE INDEX idx_revoked_tokens_user_id ON revoked_tokens(user_id);