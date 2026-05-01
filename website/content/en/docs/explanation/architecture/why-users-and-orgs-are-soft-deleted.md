---
title: Why users and orgs are soft-deleted
weight: 2
---

Migration 000015 made the `users` and `organizations` tables soft-deleted: a `DELETE` at the application layer stamps `deleted_at` rather than physically removing the row. Three reasons:

1. **Audit-trail preservation.** Audit log entries reference users by ID. If a user record vanished, those entries would either dangle or have to be rewritten.
2. **Reversibility.** "Oops, that account was important" needs an undo without a backup restore.
3. **Controlled DSGVO Art. 17 erasure.** True erasure has specific requirements (free-form fields wiped). A dedicated `HardDelete` codepath that does it deliberately is safer than a default delete that occasionally satisfies erasure.

Admin restore today is direct-DB only — see [Restore a soft-deleted user or organization](../../../how-to/operate/restore-a-soft-deleted-user-or-organization/).

## Asymmetry across tables

Only `users` and `organizations` are tombstoned. Children, employees, contracts, sections, bills, audit log entries hard-delete on `DELETE`. Identity-bearing rows need the tombstone; record-keeping rows can be physically removed without breaking the audit trail (which is preserved independently).

## Contributor rule

For the rule on writing queries that respect the soft-delete invariant (auto-scoping vs. JOIN'd tables), see [Add a database migration](../../../how-to/develop/add-a-database-migration/) and `.claude/rules/database.md`.
