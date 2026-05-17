-- 000023: switch factor backup codes from SHA-256 to bcrypt.
--
-- Backup codes were hashed with unsalted SHA-256, which is brittle
-- against an offline attack on a leaked DB dump: the 64-bit-entropy
-- Crockford-base32 codespace is small enough that a single GPU can
-- chew through it faster than is comfortable, and the lack of per-row
-- salt means rainbow-table precomputation is shared across the entire
-- factor_backup_codes table. Bcrypt at DefaultCost adds per-row salt
-- and ~100ms of work per guess — the same envelope as the user
-- password hash, which is the correct primitive for a short-lived
-- recovery secret.
--
-- We cannot rehash existing rows from SHA-256 to bcrypt (the
-- plaintext was never persisted). Instead, every existing backup
-- code is invalidated here; affected users regenerate via the
-- POST /factors/{id}/backup-codes endpoint after their next
-- successful primary-factor verify. A dual-format compatibility
-- window was rejected because keeping the weak hash alive only
-- lengthens the window in which it could be exploited.

DELETE FROM factor_backup_codes;

ALTER TABLE factor_backup_codes ALTER COLUMN code_hash TYPE TEXT;
