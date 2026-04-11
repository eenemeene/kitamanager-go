-- Prevent duplicate bill uploads: same file hash per org
CREATE UNIQUE INDEX IF NOT EXISTS idx_bill_periods_org_hash
    ON government_funding_bill_periods (organization_id, file_sha256);

-- Prevent duplicate bill uploads: same billing month per org
CREATE UNIQUE INDEX IF NOT EXISTS idx_bill_periods_org_month
    ON government_funding_bill_periods (organization_id, from_date);
