-- Revert to the case-sensitive unique index. Stored emails are left in
-- their lowercased form — the original mixed-case cannot be reliably
-- restored.
DROP INDEX IF EXISTS idx_users_email_lower;
CREATE UNIQUE INDEX idx_users_email ON users (email);
