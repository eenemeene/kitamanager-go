-- Schema integrity fixes surfaced by a schema audit pass.
--
-- Four real bugs closed here, all rooted in migration 000001:
--
-- (1) pay_plans.organization_id lacked ON DELETE behavior (Postgres
--     default NO ACTION). Deleting an organization that had pay plans
--     would fail with an FK violation even though every other org-
--     scoped table cascades cleanly. Org-delete was silently broken
--     for any org that ever configured a pay plan.
--
-- (2) government_funding_bill_periods.organization_id — same bug,
--     same fix. Any org that ever uploaded a funding bill blocked
--     its own deletion.
--
-- (3) child_attendances.recorded_by referenced users(id) NOT NULL
--     with no ON DELETE. A user who had ever recorded an attendance
--     became undeletable because users(id) is CASCADEd from many
--     other tables; the CASCADE chain would abort at this FK. Fixed
--     by SET NULL (drops NOT NULL too) so attendance history is
--     preserved with an anonymised author when users are removed —
--     this matches the pattern used on audit_logs.organization_id.
--
-- (4) government_funding_bill_periods.created_by — same bug as (3),
--     same fix for the same reason.
--
-- Plus one correctness tightening:
--
-- (5) child_attendances.status was a VARCHAR(20) with no CHECK,
--     enforced as an enum only in the service layer. Direct SQL or
--     a future code path could write garbage. Added a CHECK matching
--     the four ChildAttendanceStatus constants in internal/models/
--     child_attendance.go.
--
-- All migrations are additive (DROP + ADD of named FK constraints,
-- DROP NOT NULL via ALTER COLUMN, add CHECK) and re-entrant where
-- Postgres lets us. The constraint names are the Postgres defaults
-- assigned when migration 000001 created the FKs — they match on
-- every live DB that ran 000001.

-- (1) pay_plans.organization_id → ON DELETE CASCADE
ALTER TABLE pay_plans
    DROP CONSTRAINT pay_plans_organization_id_fkey;
ALTER TABLE pay_plans
    ADD CONSTRAINT pay_plans_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

-- (2) government_funding_bill_periods.organization_id → ON DELETE CASCADE
ALTER TABLE government_funding_bill_periods
    DROP CONSTRAINT government_funding_bill_periods_organization_id_fkey;
ALTER TABLE government_funding_bill_periods
    ADD CONSTRAINT government_funding_bill_periods_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

-- (3) child_attendances.recorded_by → ON DELETE SET NULL (drop NOT NULL first)
ALTER TABLE child_attendances
    ALTER COLUMN recorded_by DROP NOT NULL;
ALTER TABLE child_attendances
    DROP CONSTRAINT child_attendances_recorded_by_fkey;
ALTER TABLE child_attendances
    ADD CONSTRAINT child_attendances_recorded_by_fkey
    FOREIGN KEY (recorded_by) REFERENCES users(id) ON DELETE SET NULL;

-- (4) government_funding_bill_periods.created_by → ON DELETE SET NULL
ALTER TABLE government_funding_bill_periods
    ALTER COLUMN created_by DROP NOT NULL;
ALTER TABLE government_funding_bill_periods
    DROP CONSTRAINT government_funding_bill_periods_created_by_fkey;
ALTER TABLE government_funding_bill_periods
    ADD CONSTRAINT government_funding_bill_periods_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

-- (5) child_attendances.status CHECK — enum values from
--     internal/models/child_attendance.go: ChildAttendanceStatus*.
ALTER TABLE child_attendances
    ADD CONSTRAINT child_attendances_status_check
    CHECK (status IN ('present', 'absent', 'sick', 'vacation'));
