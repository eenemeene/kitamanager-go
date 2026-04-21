DROP INDEX IF EXISTS idx_audit_logs_org_ts;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS organization_id;
