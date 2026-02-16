-- Period overlap queries (used by PeriodStorer.ValidateNoOverlap)
CREATE INDEX IF NOT EXISTS idx_employee_contracts_period ON employee_contracts(employee_id, from_date, to_date);
CREATE INDEX IF NOT EXISTS idx_child_contracts_period ON child_contracts(child_id, from_date, to_date);
CREATE INDEX IF NOT EXISTS idx_budget_item_entries_period ON budget_item_entries(budget_item_id, from_date);
CREATE INDEX IF NOT EXISTS idx_gov_funding_periods_period ON government_funding_periods(government_funding_id, from_date);
CREATE INDEX IF NOT EXISTS idx_pay_plan_periods_period ON pay_plan_periods(pay_plan_id, from_date);

-- Unique constraint: one attendance per child per day
CREATE UNIQUE INDEX IF NOT EXISTS idx_child_attendances_child_date ON child_attendances(child_id, date);

-- Org+date attendance queries
CREATE INDEX IF NOT EXISTS idx_child_attendances_org_date ON child_attendances(organization_id, date);

-- Pay plan entry lookup (period + grade + step)
CREATE UNIQUE INDEX IF NOT EXISTS idx_pay_plan_entries_lookup ON pay_plan_entries(period_id, grade, step);
