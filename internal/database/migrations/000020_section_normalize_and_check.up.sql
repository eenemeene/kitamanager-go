-- 000020: case-insensitive name uniqueness + age-range CHECK on sections
--
-- Mirrors the playbook B6 used for budget items (case-insensitive
-- functional unique index) and B4 used for budget item entries
-- (CHECK constraint as the truthful gate for what the service
-- already validates).
--
-- Two changes:
--
-- 1. Replace the partial unique index from migration 000018 with a
--    functional partial index on (organization_id, lower(trim(name)))
--    WHERE deleted_at IS NULL. So "Krippe", "krippe", and " Krippe "
--    all collide. Service-layer trim is preserved so the column
--    value reads cleanly; the DB index just enforces the canonical
--    form.
--
-- 2. CHECK constraint mirroring service.validateAgeRange:
--      * each age non-negative if set
--      * min < max if both set
--    Direct UPDATEs (admin scripts, future migrations, DBA console)
--    bypass the service validation; the CHECK closes that hole.

DROP INDEX IF EXISTS idx_section_org_name;
CREATE UNIQUE INDEX idx_section_org_name
    ON sections (organization_id, lower(trim(name)))
    WHERE deleted_at IS NULL;

ALTER TABLE sections
    ADD CONSTRAINT sections_age_range_valid CHECK (
        (min_age_months IS NULL OR min_age_months >= 0)
        AND (max_age_months IS NULL OR max_age_months >= 0)
        AND (min_age_months IS NULL
             OR max_age_months IS NULL
             OR min_age_months < max_age_months)
    );
