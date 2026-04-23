-- Rollback of 000011. Drops the index, the CHECK constraint, and the
-- four new columns. Any pending_mfa rows still present become orphans,
-- but the next cleanup cycle would have swept them anyway.

DROP INDEX IF EXISTS idx_sessions_kind_expires;

ALTER TABLE sessions
    DROP CONSTRAINT IF EXISTS sessions_kind_check;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS challenge_nonce,
    DROP COLUMN IF EXISTS password_verified_at,
    DROP COLUMN IF EXISTS mfa_challenge_failures,
    DROP COLUMN IF EXISTS kind;
