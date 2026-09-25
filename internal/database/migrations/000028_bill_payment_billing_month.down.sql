DROP INDEX IF EXISTS idx_bill_payments_billing_month;

ALTER TABLE government_funding_bill_payments
    DROP COLUMN IF EXISTS billing_month;
