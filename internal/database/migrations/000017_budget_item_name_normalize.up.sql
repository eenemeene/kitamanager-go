-- 000017: case-insensitive, whitespace-insensitive uniqueness on budget item names
--
-- The original index from migration 000001 was on the raw `name`
-- column, so "Rent" and "rent" produced two different items, and so
-- did "Rent" and " Rent ". The service-layer validateRequiredName
-- already strips outer whitespace before insert, but a case-only
-- collision still slipped through and confused users (two near-
-- identical rows in the list, ambiguous in financials).
--
-- Replace with a functional unique index on lower(trim(name)) so the
-- DB itself rejects collisions regardless of case or whitespace. The
-- service trims, so the column value reads cleanly; the index just
-- enforces uniqueness on the canonical form.
--
-- This is the same playbook migration 000015 used for users.email
-- (partial functional index on lower(email)).

DROP INDEX IF EXISTS idx_budget_item_org_name;
CREATE UNIQUE INDEX idx_budget_item_org_name
    ON budget_items (organization_id, lower(trim(name)));
