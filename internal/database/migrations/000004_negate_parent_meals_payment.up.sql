-- Parent contributions are deductions (reduce government payment).
-- The payment column must be negative to match the bill storage convention.
-- Only negate rows that are currently positive to make this migration idempotent.
UPDATE government_funding_properties
SET payment = -payment
WHERE key = 'parent' AND value = 'meals' AND payment > 0;
