-- 000026: pay_plan_periods CHECK constraints
--
-- weekly_hours on this table is a DIVISOR. employeeMonthlyCost computes
--   gross = round(monthly_amount * contract_weekly_hours / period_weekly_hours)
-- so a zero here yields +Inf, and int(math.Round(+Inf)) is
-- implementation-defined in Go — on amd64 it produces math.MinInt64
-- truncated to int. No panic, no error: the wrong number simply appears
-- in /statistics/financials, the forecast, and the employee-cost
-- estimate.
--
-- Migration 000021 gave employee_contracts.weekly_hours the range check
-- (0..168) for the same class of reason. The divisor never got the
-- equivalent, so the invariant lived only in
-- validatePayPlanPeriodFields — an application-path guard that a direct
-- SQL UPDATE, an admin script, or a future code path that forgets to
-- call it writes straight past. Same playbook as 000014
-- (child_attendances), 000016 (budget_item_entries) and 000021.
--
-- Bounds mirror validatePayPlanPeriodFields exactly so the two cannot
-- drift:
--   * weekly_hours strictly > 0 (not >= 0 like employee_contracts —
--     a contract may legitimately have zero hours during parental
--     leave, but a pay plan period with zero hours cannot define a
--     full-time basis to pro-rate against)
--   * weekly_hours <= 168 (validation.MaxWeeklyHours)
--   * employer_contribution_rate in [0, 10000] hundredths of a percent
--
-- Both columns are NOT NULL, so the CHECKs apply to every row.
--
-- Deliberately a plain ADD CONSTRAINT rather than NOT VALID + VALIDATE:
-- a row violating this has been producing wrong money, so a migration
-- that stops and names it is the outcome we want. If this fails on a
-- live database, the offending pay plan period must be corrected before
-- the deploy proceeds — the numbers it fed were never trustworthy.

ALTER TABLE pay_plan_periods
    ADD CONSTRAINT pay_plan_periods_weekly_hours_valid
    CHECK (weekly_hours > 0 AND weekly_hours <= 168);

ALTER TABLE pay_plan_periods
    ADD CONSTRAINT pay_plan_periods_employer_contribution_rate_valid
    CHECK (employer_contribution_rate >= 0 AND employer_contribution_rate <= 10000);
