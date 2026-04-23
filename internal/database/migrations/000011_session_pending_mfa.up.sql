-- Two-step login state lives on the existing sessions table via a `kind`
-- discriminator. `regular` rows are the ordinary authenticated sessions
-- from migration 000008; `pending_mfa` rows are the short-lived
-- intermediate state between a valid password and a verified MFA code.
--
-- The pending-row columns are nullable on purpose: old `regular` rows
-- created before this migration keep NULL for them, and the app layer
-- never reads those columns on regular sessions.
--
-- activation_failures on factors is the per-row brute-force cap during
-- enrollment; mfa_challenge_failures here is the analogous cap at
-- login-time. When it reaches the service-layer limit, the pending row
-- is deleted and the user restarts with the password step.
--
-- challenge_nonce is a nullable BYTEA reserved for future WebAuthn
-- integration (the server-issued challenge bound to a single
-- navigator.credentials.get() ceremony). TOTP and backup_codes never
-- populate it. Keeping the column here instead of behind a later
-- migration means the factor-generic verify code path stays one shape
-- across future factor types.

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS kind VARCHAR(32) NOT NULL DEFAULT 'regular',
    ADD COLUMN IF NOT EXISTS mfa_challenge_failures BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS password_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS challenge_nonce BYTEA;

-- Tighten `kind` to the values the app layer knows how to reason about.
-- Using IF NOT EXISTS would be nice but Postgres < 17 doesn't support
-- it on ADD CONSTRAINT — the ALTER TABLE above is re-entrant, and this
-- constraint add is guarded by an inline DO block so the migration can
-- re-run cleanly on a partially-applied database.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sessions_kind_check'
    ) THEN
        ALTER TABLE sessions
            ADD CONSTRAINT sessions_kind_check
            CHECK (kind IN ('regular', 'pending_mfa'));
    END IF;
END$$;

-- Cleanup job + per-user lookup filter both key off (kind, expires_at).
CREATE INDEX IF NOT EXISTS idx_sessions_kind_expires ON sessions(kind, expires_at);
