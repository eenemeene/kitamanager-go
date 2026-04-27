-- 000019: enforce one default section per organization at the DB layer
--
-- Today nothing prevents two sections in the same org from both
-- having `is_default = true`. The seed/org-create path
-- (CreateWithDefaultSection) is the only writer in the application,
-- so the invariant has held by convention. But: a future bug, a
-- direct admin SQL, or a future "promote to default" flow that
-- forgets to flip the previous one would silently leave the system
-- with two defaults — and `FindDefaultSection` returns whichever
-- comes back first (non-deterministic).
--
-- A partial unique index makes the invariant truthful:
--   * Only the "live default" rows are constrained.
--   * Non-default rows (the vast majority) and tombstoned rows are
--     not in the index, so it stays tiny.
--
-- The companion service change adds a `PromoteToDefault` operation
-- that runs the flag-flip in a single UPDATE — see service/section.go.

CREATE UNIQUE INDEX IF NOT EXISTS idx_section_one_default_per_org
    ON sections (organization_id)
    WHERE is_default = true AND deleted_at IS NULL;
