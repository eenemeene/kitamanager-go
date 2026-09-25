-- Restores the index 000028 created, so a down migration to 28 leaves the
-- schema 28 describes. It is not useful -- see the up migration for the
-- measurements -- but "down" means "put it back", not "keep the improvement".

CREATE INDEX IF NOT EXISTS idx_bill_payments_billing_month
    ON government_funding_bill_payments (billing_month);
