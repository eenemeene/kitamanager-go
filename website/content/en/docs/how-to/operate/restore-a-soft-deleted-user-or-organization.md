---
title: Restore a soft-deleted user or organization
weight: 8
---

You deleted a user (or organisation) and you need them back. Possible — KitaManager **soft-deletes** users and organisations, so the row is still in the database, just hidden.

{{< callout type="warning" >}}
There is **no UI or API endpoint** for restore today. This recipe requires direct database access. The "admin trash view" mentioned in some comments is planned but not yet implemented.
{{< /callout >}}

## Find the row

Connect to the Postgres database (Docker Compose: `docker compose exec -it postgres psql -U $DB_USER $DB_NAME`).

```sql
-- Find a soft-deleted user
SELECT id, name, email, deleted_at FROM users WHERE deleted_at IS NOT NULL;

-- Find a soft-deleted organisation
SELECT id, name, deleted_at FROM organizations WHERE deleted_at IS NOT NULL;
```

## Restore the row

```sql
UPDATE users SET deleted_at = NULL WHERE id = <ID>;
-- or
UPDATE organizations SET deleted_at = NULL WHERE id = <ID>;
```

The user can sign in again immediately. The organisation reappears for every member.

## What about other tables?

Only `users` and `organizations` are soft-deleted. Children, employees, contracts, sections, bills — all hard-delete on `DELETE`. **If you deleted any of those by mistake, the only recovery is a database restore from backup.** See [Back up and restore](../back-up-and-restore/).

## Why no admin UI yet

The pattern was added in migration 000015 to support a future admin trash-view; the API + UI haven't been built. If your team frequently restores deleted users, prioritise the feature.

## Notes

- The audit log records the original delete event. After restore, optionally write a manual note (e.g. another audit entry, or a comment in your team's ticket tracker) explaining why the restore happened.
- Restoring a user does not restore their organisation memberships — those are stored in a separate table that hard-deletes. Re-add memberships via [Assign a role in an organization](../../administer/assign-role-in-organization/).
