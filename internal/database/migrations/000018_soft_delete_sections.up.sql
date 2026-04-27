-- 000018: soft-delete for sections
--
-- Why this lands now: HasChildren / HasEmployees today count
-- HISTORICAL contracts (no time filter), so a section with only
-- ENDED contracts could never be deleted — orgs that reorganise
-- their sections accumulate zombie sections forever, and the UI
-- gives the user no path out. Two pieces fix that:
--
--   1. (this migration) Soft-delete on `sections`. Tombstoned
--      sections disappear from List / pickers, but their `id`
--      still satisfies the contract.section_id FK so historical
--      child_contracts and employee_contracts keep resolving.
--
--   2. (companion service change) HasChildren / HasEmployees get
--      time-filtered to "active on now" + use EXISTS-via-LIMIT-1
--      semantics. Only currently-assigned contracts block delete.
--
-- Same playbook migration 000015 used for users + organizations.

ALTER TABLE sections
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Sparse index for the future retention cleanup job. Most sections
-- never get tombstoned, so the partial keeps it tiny.
CREATE INDEX IF NOT EXISTS idx_sections_deleted_at
    ON sections(deleted_at)
    WHERE deleted_at IS NOT NULL;

-- Swap the existing per-org name unique index for a partial
-- predicate so a soft-deleted section's name can be reused. (The
-- companion S5 migration replaces this again with a functional
-- case-insensitive variant; doing both rewrites here would mix
-- concerns and obscure the soft-delete intent.)
DROP INDEX IF EXISTS idx_section_org_name;
CREATE UNIQUE INDEX idx_section_org_name
    ON sections (organization_id, name)
    WHERE deleted_at IS NULL;
