-- Optimistic concurrency for contracts.
--
-- Contract writes had no concurrency control of any kind: no version column, no
-- ETag/If-Match, and the stores called a bare GORM Save() without inspecting
-- RowsAffected. Two people editing the same contract silently last-write-wins,
-- and because a contract's care type and supplements determine its funding, a
-- lost update quietly changes money.
--
-- Every write bumps `version`, and updates are guarded with
-- `WHERE id = ? AND version = ?` so a stale writer affects zero rows and can be
-- reported rather than overwriting the newer state.
--
-- Existing rows start at 1. NOT NULL DEFAULT 1 means older clients that do not
-- send a precondition keep working during the migration window.

ALTER TABLE child_contracts
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;

ALTER TABLE employee_contracts
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;
