-- Replace JWT + refresh-token + RevokedToken sentinel machinery with opaque
-- server-side sessions. The session id stored here is sha256(cookie_value);
-- the raw value only ever lives in the client cookie, so a DB read-leak does
-- not expose usable credentials.

CREATE TABLE IF NOT EXISTS sessions (
    id CHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_ip VARCHAR(45),
    created_user_agent TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

DROP TABLE IF EXISTS revoked_tokens;
