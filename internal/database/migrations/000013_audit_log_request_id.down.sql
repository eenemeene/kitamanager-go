DROP INDEX IF EXISTS idx_audit_logs_request_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS request_id;
