---
title: Assign a role in an organization
weight: 5
---

You want to give a user access to your organisation, or change their role.

## Steps

1. Open **Settings** → **Users** → click the user.
2. In the **Organization memberships** section, click **Add organization** (for a new membership) or click an existing organisation row to change the role.
3. Pick the organisation and the role: `admin`, `manager`, `member`, or `staff`. (`superadmin` is global, not org-scoped — see below.)
4. Click **Save**.

The user's permissions update immediately on their next request.

## Notes

- For the role definitions, see [Reference: RBAC](../../../reference/rbac/).
- A user can be a member of **multiple** organisations with a different role in each. Use this for a person who manages two Kitas.
- Granting **superadmin** is a separate, dedicated control on the user detail page (only existing superadmins can grant it). It's global, not per-organisation. See [User Management Reference](../../../reference/rbac/) for what superadmin grants.
- Audit log records the assignment.
