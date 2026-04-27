-- 000017 down: restore the raw-name unique index from 000001.
--
-- Note: this only succeeds if no two existing rows differ only by
-- case or whitespace under the new normalisation. If they do,
-- restoring the simpler index would be a uniqueness violation —
-- the rollback is best-effort and may fail.

DROP INDEX IF EXISTS idx_budget_item_org_name;
CREATE UNIQUE INDEX idx_budget_item_org_name
    ON budget_items (organization_id, name);
