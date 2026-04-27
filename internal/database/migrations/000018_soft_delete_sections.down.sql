-- 000018 down: restore the original raw-name unique index from 000001
-- and drop the soft-delete column. Will fail if any tombstoned rows
-- exist whose names collide with live rows under the old (non-partial)
-- index — the rollback is best-effort.

DROP INDEX IF EXISTS idx_section_org_name;
DROP INDEX IF EXISTS idx_sections_deleted_at;

CREATE UNIQUE INDEX idx_section_org_name
    ON sections (organization_id, name);

ALTER TABLE sections
    DROP COLUMN IF EXISTS deleted_at;
