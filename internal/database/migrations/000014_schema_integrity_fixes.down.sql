-- Reverse of 000014. Drops the CHECK, restores the NO-ACTION /
-- NOT NULL semantics that 000001 shipped with. Restoring the broken
-- state is deliberately unpleasant — it matches production reality
-- for any DB that has not yet run 000014, so this down-migration is
-- only useful for local test reset, not for rolling back a
-- completed production upgrade.

ALTER TABLE child_attendances
    DROP CONSTRAINT child_attendances_status_check;

ALTER TABLE government_funding_bill_periods
    DROP CONSTRAINT government_funding_bill_periods_created_by_fkey;
ALTER TABLE government_funding_bill_periods
    ADD CONSTRAINT government_funding_bill_periods_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users(id);
ALTER TABLE government_funding_bill_periods
    ALTER COLUMN created_by SET NOT NULL;

ALTER TABLE child_attendances
    DROP CONSTRAINT child_attendances_recorded_by_fkey;
ALTER TABLE child_attendances
    ADD CONSTRAINT child_attendances_recorded_by_fkey
    FOREIGN KEY (recorded_by) REFERENCES users(id);
ALTER TABLE child_attendances
    ALTER COLUMN recorded_by SET NOT NULL;

ALTER TABLE government_funding_bill_periods
    DROP CONSTRAINT government_funding_bill_periods_organization_id_fkey;
ALTER TABLE government_funding_bill_periods
    ADD CONSTRAINT government_funding_bill_periods_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id);

ALTER TABLE pay_plans
    DROP CONSTRAINT pay_plans_organization_id_fkey;
ALTER TABLE pay_plans
    ADD CONSTRAINT pay_plans_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id);
