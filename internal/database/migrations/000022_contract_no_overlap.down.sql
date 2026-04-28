-- The btree_gist extension stays installed (other tables already rely on it
-- via migration 000016) — same convention as that migration's down-script.
ALTER TABLE employee_contracts DROP CONSTRAINT IF EXISTS employee_contracts_no_overlap;
ALTER TABLE child_contracts DROP CONSTRAINT IF EXISTS child_contracts_no_overlap;
