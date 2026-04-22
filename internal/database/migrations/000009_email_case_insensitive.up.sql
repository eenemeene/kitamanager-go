-- Normalize stored user emails to lowercase and enforce case-insensitive
-- uniqueness via a functional index. Prior to this migration the unique
-- index was on `email` directly (case-sensitive), so two rows that differ
-- only in case could technically coexist. This closes that gap.

-- Safety gate: if two rows already collide when lowercased, we cannot
-- auto-fix without deleting one. Abort with a clear error so an operator
-- deduplicates manually before re-running.
DO $$
DECLARE
    dup_count INT;
BEGIN
    SELECT COUNT(*) INTO dup_count FROM (
        SELECT lower(email) FROM users GROUP BY lower(email) HAVING COUNT(*) > 1
    ) AS dups;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'Cannot normalize email case: % case-variant duplicate(s) exist. '
            'Deduplicate rows where lower(email) collides before re-running this migration.',
            dup_count;
    END IF;
END $$;

-- Normalize existing rows. Safe now that we've verified no collisions.
UPDATE users SET email = lower(email) WHERE email <> lower(email);

-- Swap the case-sensitive unique index for a case-insensitive functional one.
DROP INDEX IF EXISTS idx_users_email;
CREATE UNIQUE INDEX idx_users_email_lower ON users (lower(email));
