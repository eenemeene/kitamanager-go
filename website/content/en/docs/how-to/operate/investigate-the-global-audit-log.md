---
title: Investigate the global audit log
weight: 6
---

You're a superadmin and you want to investigate something across organisations, or look at login/auth events that the org-scoped audit log hides.

The global audit log is **API-only** today. There's no dedicated UI page; superadmins query the API directly.

## Steps

```bash
# All events between two dates
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs?from=2026-04-01&to=2026-04-30"

# Filter by action substring (e.g. all logins)
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs?action=login"

# Filter by actor user ID
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs?user_id=42"

# Get a specific entry
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs/12345"
```

For the request/response details, see [API: Audit logs](../../../reference/api/#audit-logs).

## What's in the global log that's not in the org-scoped log

- `login_success`, `login_failed` (with the IP that attempted)
- `password_change_self`, `password_change_admin`
- `mfa_factor_create`, `mfa_factor_delete`, `mfa_verify_failed`
- Cross-organisation actions (org create/delete, superadmin grant/revoke, funding-rate updates)

## Notes

- The org-scoped audit log at `/api/v1/organizations/{orgId}/audit-logs` *deliberately* hides login/password events because they are sensitive across organisations. A user who is admin in two Kitas should not learn from one org's log when the same user signed in for the other org.
- Audit entries are append-only. There is no API to modify or delete them.
- For routine audit queries (who deleted what in our org last week), the org-scoped log via the UI is enough — see [Review the org audit log](../../administer/review-audit-log/).
