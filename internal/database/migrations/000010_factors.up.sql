-- Multi-factor authentication primitives. The shape is factor-generic so
-- WebAuthn / passkeys can be added later as additional `type` values +
-- one new subtable per type.
--
-- The parent `factors` table holds everything common to every factor
-- type. Type-specific fields live in subtables joined 1:1 on factor_id.
-- Partial unique index enforces "at most one backup_codes factor per
-- user" at the DB layer.

CREATE TABLE IF NOT EXISTS factors (
    id                   BIGSERIAL   PRIMARY KEY,
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type                 VARCHAR(32) NOT NULL,
    label                VARCHAR(100),
    enabled_at           TIMESTAMPTZ,                    -- NULL while enrollment is pending
    last_used_at         TIMESTAMPTZ,
    -- activation_failures caps how many wrong codes a user (or attacker
    -- inside their session) can try against a pending factor. When it
    -- reaches FactorActivationFailureLimit (service-layer constant) the
    -- row is auto-deleted and the user must re-enrol, closing the
    -- otherwise-trivial TOTP brute-force surface against a pending row.
    activation_failures  BIGINT      NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL,
    CONSTRAINT factors_type_check CHECK (type IN ('totp', 'backup_codes'))
);
CREATE INDEX IF NOT EXISTS idx_factors_user_id ON factors(user_id);
CREATE INDEX IF NOT EXISTS idx_factors_user_enabled ON factors(user_id, enabled_at)
    WHERE enabled_at IS NOT NULL;

-- Singleton invariant: exactly one backup_codes factor per user.
CREATE UNIQUE INDEX IF NOT EXISTS idx_factors_user_singleton_backup
    ON factors(user_id) WHERE type = 'backup_codes';

-- TOTP-specific subtable. Secret is AES-256-GCM at rest, keyed by the
-- TOTP_ENCRYPTION_KEY env var. last_used_step is the RFC 6238 time-step
-- count of the most recently accepted code; an atomic compare-and-set
-- UPDATE on this column is how replay is prevented.
CREATE TABLE IF NOT EXISTS factor_totp_secrets (
    factor_id         BIGINT PRIMARY KEY REFERENCES factors(id) ON DELETE CASCADE,
    secret_ciphertext BYTEA  NOT NULL,
    secret_nonce      BYTEA  NOT NULL,
    last_used_step    BIGINT
);

-- Backup codes subtable. One row per individual code; code_hash is
-- sha256 of the raw (user-facing) string. used_at is the atomic
-- single-use flag — the verify path uses
--   UPDATE ... SET used_at = NOW() WHERE id = $1 AND used_at IS NULL
-- so that two concurrent attempts against the same code can't both
-- succeed.
CREATE TABLE IF NOT EXISTS factor_backup_codes (
    id         BIGSERIAL PRIMARY KEY,
    factor_id  BIGINT NOT NULL REFERENCES factors(id) ON DELETE CASCADE,
    code_hash  CHAR(64) NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_factor_backup_codes_factor_id ON factor_backup_codes(factor_id);
