-- 000020 down: restore the raw-name partial unique index from 000018
-- and drop the age CHECK. Will fail if any existing rows differ only
-- by case/whitespace under the functional normalisation, or if any
-- row has an out-of-range age — best-effort rollback.

ALTER TABLE sections
    DROP CONSTRAINT IF EXISTS sections_age_range_valid;

DROP INDEX IF EXISTS idx_section_org_name;
CREATE UNIQUE INDEX idx_section_org_name
    ON sections (organization_id, name)
    WHERE deleted_at IS NULL;
