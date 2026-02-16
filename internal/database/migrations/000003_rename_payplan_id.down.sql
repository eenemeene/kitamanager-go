ALTER TABLE employee_contracts RENAME COLUMN pay_plan_id TO payplan_id;
DROP INDEX IF EXISTS idx_employee_contracts_pay_plan_id;
CREATE INDEX IF NOT EXISTS idx_employee_contracts_payplan_id ON employee_contracts(payplan_id);
