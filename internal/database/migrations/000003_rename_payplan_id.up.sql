-- Rename payplan_id to pay_plan_id to match GORM's default column naming convention.
-- GORM converts PayPlanID -> pay_plan_id (snake_case with underscores).

ALTER TABLE employee_contracts RENAME COLUMN payplan_id TO pay_plan_id;
DROP INDEX IF EXISTS idx_employee_contracts_payplan_id;
CREATE INDEX IF NOT EXISTS idx_employee_contracts_pay_plan_id ON employee_contracts(pay_plan_id);
