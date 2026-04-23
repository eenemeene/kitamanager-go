-- WebAuthn / FIDO2 factor support. Adds a third value to the factor
-- type discriminator, a pair of registration-challenge columns on the
-- parent factors row, and a type-specific subtable mirroring the
-- pattern established in migration 000010 for totp and backup_codes.
--
-- Why the two-step enrolment model needs a registration challenge on
-- the parent row:
--   * POST /factors creates a pending factor and MUST hand back a
--     server-generated challenge (>=16 bytes random) inside the
--     PublicKeyCredentialCreationOptions. The challenge is single-use
--     and short-lived.
--   * POST /factors/:id/activate receives the attestation object, must
--     re-read the challenge, and the WebAuthn library verifies that
--     the attestation was signed over exactly that challenge.
-- Storing the challenge on the pending factor row keeps the lifecycle
-- symmetric with TOTP (the totp secret also lives only until activate
-- fires); a separate table would just double the GC surface.
--
-- Why a unique index on credential_id:
--   WebAuthn L3 section 7.1 step 22 requires the RP to reject a
--   registration whose credential_id is already bound to any user.
--   Enforcing it at the index layer means a concurrent double-submit
--   races cleanly into a unique-violation rather than leaving two
--   rows pointing at the same hardware key.

ALTER TABLE factors DROP CONSTRAINT factors_type_check;
ALTER TABLE factors
    ADD CONSTRAINT factors_type_check
    CHECK (type IN ('totp', 'backup_codes', 'webauthn'));

ALTER TABLE factors
    ADD COLUMN IF NOT EXISTS registration_challenge BYTEA,
    ADD COLUMN IF NOT EXISTS registration_challenge_expires_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS factor_webauthn_credentials (
    factor_id           BIGINT       PRIMARY KEY REFERENCES factors(id) ON DELETE CASCADE,
    credential_id       BYTEA        NOT NULL,
    -- COSE-encoded public key. Plaintext at rest — it's a public key
    -- by definition and storing it encrypted would just confuse
    -- operators later.
    public_key          BYTEA        NOT NULL,
    -- AAGUID identifies the authenticator model. Under attestation=none
    -- browsers zero this out, but we store it regardless so the UI can
    -- surface the hardware where known. 16 bytes fixed.
    aaguid              BYTEA,
    -- WebAuthn L3 clone-detection counter. Many synced-passkey
    -- authenticators always report 0; the verify path soft-warns on a
    -- regression rather than failing (see service/factor.go).
    sign_count          BIGINT       NOT NULL DEFAULT 0,
    -- Comma-joined transport hints ("usb,nfc,ble,hybrid,internal"),
    -- fed back to the browser in allowCredentials[] so it can pre-
    -- filter authenticators before prompting the user.
    transports          VARCHAR(255),
    attestation_format  VARCHAR(64),
    -- Backup Eligible: permanent, set at registration; true means the
    -- credential CAN be synced off the device.
    backup_eligible     BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Backup State: mutable, updated on every assertion; true means the
    -- credential is CURRENTLY synced.
    backup_state        BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Whether the authenticator verified the user (PIN/biometric) at
    -- registration. Consulted only if a future step-up flow needs a
    -- UV-proven factor.
    uv_initialized      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL
);

-- Unique across the whole table: credential_id identifies the
-- authenticator globally and must not collide across users.
CREATE UNIQUE INDEX IF NOT EXISTS idx_factor_webauthn_credential_id
    ON factor_webauthn_credentials(credential_id);
