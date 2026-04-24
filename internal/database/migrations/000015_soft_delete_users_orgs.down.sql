-- Reverse of 000015. Drops the deleted_at columns and restores the
-- full (non-partial) unique indexes that 000009/000001 shipped.
--
-- WARNING: any rows with non-NULL deleted_at will be resurrected as
-- "live" rows by this revert. If two tombstoned rows share an email
-- / org name, the DROP COLUMN + recreated unique index will fail.
-- Purge tombstones before running this down migration if that
-- situation applies.

DROP INDEX IF EXISTS idx_organizations_name;
CREATE UNIQUE INDEX idx_organizations_name ON organizations(name);
DROP INDEX IF EXISTS idx_organizations_deleted_at;
ALTER TABLE organizations DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_users_email_lower;
CREATE UNIQUE INDEX idx_users_email_lower ON users (lower(email));
DROP INDEX IF EXISTS idx_users_deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
