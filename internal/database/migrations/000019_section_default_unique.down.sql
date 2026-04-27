-- 000019 down: drop the partial unique index. Live data is unaffected.

DROP INDEX IF EXISTS idx_section_one_default_per_org;
