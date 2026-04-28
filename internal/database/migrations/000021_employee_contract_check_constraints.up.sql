-- 000021: harden employee_contracts with DB-level CHECK constraints
--
-- The service layer in internal/service/employee_contract.go validates
--   * staff_category ∈ {qualified, supplementary, non_pedagogical}
--   * 0 <= step <= 10
--   * 0 <= weekly_hours <= 168
-- but those guards run only on the application path. A direct SQL UPDATE
-- (admin scripts, future migrations, DBA console) writes garbage without
-- complaint, then surfaces months later as a NaN salary or a missing
-- pay-plan entry. Same playbook migration 000014 used for child_attendances
-- and migration 000016 used for budget_item_entries.
--
-- All three CHECKs treat NULL as passing per standard SQL semantics, which
-- matches the existing nullability of step / weekly_hours and is fine
-- because the application never writes NULL into them (Go zero values
-- become 0, not NULL). staff_category is NOT NULL with a default already,
-- so the IN-list applies on every row.

ALTER TABLE employee_contracts
    ADD CONSTRAINT employee_contracts_staff_category_valid
    CHECK (staff_category IN ('qualified', 'supplementary', 'non_pedagogical'));

ALTER TABLE employee_contracts
    ADD CONSTRAINT employee_contracts_step_valid
    CHECK (step IS NULL OR (step >= 0 AND step <= 10));

ALTER TABLE employee_contracts
    ADD CONSTRAINT employee_contracts_weekly_hours_valid
    CHECK (weekly_hours IS NULL OR (weekly_hours >= 0 AND weekly_hours <= 168));
