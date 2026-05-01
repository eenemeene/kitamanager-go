---
title: Edit or deactivate a user
weight: 3
---

You want to update a user's name/email or stop them from signing in without deleting their record.

## Edit name or email

1. Open **Settings** → **Users** → click the user.
2. Update **Name** and/or **Email** and click **Save**.

The change is recorded in the audit log.

## Deactivate a user

Untick **Active** and save. An inactive user cannot sign in. Their existing data (audit log entries, contracts they created) is preserved.

## Delete a user

Click **Delete** on the user's detail page. Their organisation memberships are removed and they cannot sign in. The user record is **soft-deleted**: hidden from regular reads but still in the database, so it can be restored without a backup.

## Notes

- Prefer **deactivate** over **delete** when someone leaves: deactivation preserves the audit trail and the ability to reactivate without DB access.
- Restoring a deleted user is possible but requires direct database access today — see [Restore a soft-deleted user or organization](../../operate/restore-a-soft-deleted-user-or-organization/). There is no admin UI for this yet.
