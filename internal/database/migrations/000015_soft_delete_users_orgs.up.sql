-- Soft-delete (Phase 1) for the two identity-level tables: users and
-- organizations. DELETE at the app layer becomes a timestamped
-- tombstone; hard-delete stays available via an explicit purge path
-- for the Art. 17 compliance story and retention-TTL cleanup.
--
-- Why this is a deliberately narrow first step:
--   - users + organizations are the two tables where undo has a
--     clear product value and where every downstream path already
--     looks them up by id.
--   - children / employees / pay_plans have their own retention
--     windows (German AO § 147 mandates 10y for financial records)
--     and index surfaces; tackling them in a follow-up keeps the
--     blast radius here small.
--   - Every Log* method on AuditService already threads ctx (PR
--     #155), so the new soft-delete / hard-delete audit events slot
--     in cleanly.
--
-- Index migration: simple partial unique indexes allow reuse of
-- email / org name after a soft-delete. Postgres cannot add a
-- partial predicate to an existing index in-place, so we drop and
-- recreate. The recreate happens in the same migration file, so the
-- window without enforcement is the runtime of this migration
-- (seconds) — deploy at a low-traffic moment. A future hardening
-- pass can switch to CREATE INDEX CONCURRENTLY.

-- ------------------------------------------------------------------
-- users
-- ------------------------------------------------------------------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Partial index used by the retention cleanup job ("find rows past
-- TTL"). NULL rows are not stored in the index, keeping it tight.
CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users(deleted_at)
    WHERE deleted_at IS NOT NULL;

-- Swap the functional case-insensitive unique index (introduced in
-- 000009) for a partial functional unique that only constrains live
-- rows. A soft-deleted row's email is therefore reusable — a new
-- user can register with the same email after the old row is
-- tombstoned.
DROP INDEX IF EXISTS idx_users_email_lower;
CREATE UNIQUE INDEX idx_users_email_lower
    ON users (lower(email))
    WHERE deleted_at IS NULL;

-- ------------------------------------------------------------------
-- organizations
-- ------------------------------------------------------------------
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_organizations_deleted_at
    ON organizations(deleted_at)
    WHERE deleted_at IS NOT NULL;

-- Swap the plain unique index on organization name for a partial.
DROP INDEX IF EXISTS idx_organizations_name;
CREATE UNIQUE INDEX idx_organizations_name
    ON organizations (name)
    WHERE deleted_at IS NULL;
