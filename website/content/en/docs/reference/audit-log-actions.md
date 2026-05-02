---
title: Audit log action codes
weight: 6
---

Every audit log entry carries an `action` field. These are the values it can take. Use them when filtering the audit log via the UI's **Action** field or via the API's `?action=…` parameter (substring match).

## Authentication and sessions

| Code | When emitted |
|---|---|
| `login` | Successful sign-in (after MFA if enabled). |
| `login_failed` | Wrong password or unknown email. |
| `login_mfa_required` | First step succeeded; awaiting MFA. |
| `logout` | User signed out. |
| `session_revoked` | A session was revoked (by the user or by another sign-in). |

## Multi-factor authentication

| Code | When emitted |
|---|---|
| `factor_enrolled` | TOTP or WebAuthn factor activated. |
| `factor_deleted` | User removed one of their own factors. |
| `factor_admin_deleted` | An admin removed a user's factor. |
| `factor_label_updated` | Renamed a factor (e.g. "Yubikey-blue"). |
| `factor_activation_locked` | Too many wrong codes during enrolment. |
| `backup_codes_regenerated` | User regenerated their recovery codes. |
| `mfa_challenge_succeeded` | TOTP / backup code / WebAuthn step passed. |
| `mfa_challenge_failed` | MFA verification failed. |
| `mfa_challenge_locked` | Too many wrong MFA attempts; account temporarily locked. |

## Passwords

| Code | When emitted |
|---|---|
| `password_change` | Self-service password change succeeded. |
| `password_change_failed` | Self-service change rejected (wrong current password, etc.). |
| `password_reset` | An admin reset another user's password. |
| `password_reset_failed` | Admin reset rejected on the `actor_password` step-up. `user_id` is the actor; `resource_id` is the target. Drives the per-actor lockout. |

## Users and organisations

| Code | When emitted |
|---|---|
| `user_create` | New user account created. |
| `user_delete` | User soft-deleted. |
| `user_purged` | User hard-deleted (DSGVO Art. 17 erasure or TTL purge). |
| `user_add_to_org` | User assigned a role in an organisation. |
| `user_remove_from_org` | Membership removed. |
| `role_change` | Role within an organisation changed. |
| `superadmin_grant` | Superadmin status granted. |
| `superadmin_revoke` | Superadmin status revoked. |
| `superadmin_change_failed` | Superadmin grant/revoke rejected on the `actor_password` step-up. `user_id` is the actor; `resource_id` is the target. |
| `org_create` | Organisation created (superadmin only). |
| `org_delete` | Organisation soft-deleted. |
| `org_purged` | Organisation hard-deleted. |

## System

| Code | When emitted |
|---|---|
| `audit_log_purged` | Retention TTL swept old audit rows. `details` carries `deleted_rows` and the `older_than` cutoff. |

## Resources

The mutating operations on most resources emit an action of the form `<resource>_<verb>` — `child_create`, `child_update`, `child_delete`, `employee_create`, `government_funding_bill_create`, etc. Substring-filter on the resource name (`child`, `employee`, `government_funding_bill`) to see them all.

`employee_delete` and `child_delete` are listed explicitly above because they are the most-checked entries during a triage.

## Notes

- The org-scoped audit log at `Settings → Audit log` deliberately excludes login/password/MFA events because they are sensitive across organisations. Superadmins see them via `GET /api/v1/audit-logs` — see [Investigate the global audit log](../../how-to/operate/investigate-the-global-audit-log/).
- The list above is the snapshot at time of writing; the source of truth is `internal/models/audit.go`.
