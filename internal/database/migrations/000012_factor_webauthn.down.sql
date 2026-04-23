-- Rollback of 000012. Drops the subtable, the extra columns on
-- factors, and restores the pre-WebAuthn CHECK constraint. Any
-- existing webauthn rows are destroyed along with the table; operators
-- running this in production must reckon with that.

DROP TABLE IF EXISTS factor_webauthn_credentials;

ALTER TABLE factors
    DROP COLUMN IF EXISTS registration_challenge_expires_at,
    DROP COLUMN IF EXISTS registration_challenge;

ALTER TABLE factors DROP CONSTRAINT IF EXISTS factors_type_check;
-- Destroy any webauthn factor rows before narrowing the CHECK.
DELETE FROM factors WHERE type = 'webauthn';
ALTER TABLE factors
    ADD CONSTRAINT factors_type_check
    CHECK (type IN ('totp', 'backup_codes'));
