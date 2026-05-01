---
title: Review the org audit log
weight: 8
---

You want to see who changed what in your organisation, and when.

## Steps

1. Open **Settings** → **Audit log** in the sidebar.
2. The table shows time, user, action (e.g. `child_create`, `section_delete`), affected resource, IP address, and result.
3. Filter by date range, or type a substring into **Action** (e.g. `delete` matches every delete action).

## Notes

- The org audit log is **append-only**. Entries cannot be edited or deleted from the UI.
- Login and password events are intentionally **excluded** from the org-scoped log because they're cross-organisational. Superadmins can see them via the API — see [Investigate the global audit log](../../operate/investigate-the-global-audit-log/).
- For the per-field schema, see [API: Audit logs](../../../reference/api/#audit-logs).
- Common detective scenarios:
  - "Who deleted that child?" — filter by action `child_delete`.
  - "What changed yesterday?" — filter by date range.
  - "Did anyone touch a Bescheid?" — filter by action `government_funding_bill_*`.
