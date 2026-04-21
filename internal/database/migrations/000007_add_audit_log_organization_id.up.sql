-- Make the audit log multi-tenant aware. Org admins will read the subset
-- of rows that carry their organization_id; identity-level events (login,
-- password, superadmin) stay NULL on purpose and remain visible only to
-- the superadmin-only global endpoint.
--
-- ON DELETE SET NULL keeps the row if an org is ever removed — the audit
-- trail must never be destroyed by a cascade.
ALTER TABLE audit_logs
    ADD COLUMN organization_id BIGINT
    REFERENCES organizations(id) ON DELETE SET NULL;

-- Backfill for the existing single-org production deployment. The subquery
-- associates every non-identity-level event with the lone org. If this
-- migration ever runs against a multi-org database, the first org wins —
-- which is the reason we do NOT want this migration to run there. New
-- environments that reach the multi-org stage must handle any pre-existing
-- global rows explicitly.
UPDATE audit_logs
   SET organization_id = (SELECT id FROM organizations ORDER BY id LIMIT 1)
 WHERE action NOT IN (
       'login',
       'login_failed',
       'logout',
       'superadmin_grant',
       'superadmin_revoke',
       'password_change',
       'password_change_failed',
       'password_reset',
       'user_create',
       'user_update',
       'user_delete'
   );

CREATE INDEX IF NOT EXISTS idx_audit_logs_org_ts
    ON audit_logs(organization_id, timestamp DESC);
