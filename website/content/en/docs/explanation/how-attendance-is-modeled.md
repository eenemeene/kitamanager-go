---
title: How attendance is modeled
weight: 5
---

Attendance in KitaManager is per-child, per-day. This page explains the data model so the UI's behaviour ("why does this row appear here?", "what does empty mean?") makes sense.

## The model

For each (child, date) pair, there is at most one **attendance record**. The record carries a status — currently `present` or `absent`. Absence of a record means **no observation has been made**, not "the child wasn't here".

This three-state model (present / absent / no record) is important. A blank cell in the weekly grid is *not* an unmarked absence — it's "we don't know". Reports that count "absent days" only count rows with `absent` status; rows that simply don't exist don't enter either count.

## Active-contract scoping

The weekly attendance grid only lists children whose care contract is active during the displayed week. A child whose contract starts next month doesn't appear — recording attendance for them before they're enrolled is a data error the model prevents.

If you change a child's contract from/to dates, the attendance grid reflects the change immediately on the next refresh.

## Auto-save

The grid auto-saves on every cell change. There's no Save button. The implementation:

1. Click a cell to cycle through the states.
2. The frontend calls the create/update/delete endpoint depending on the state transition.
3. On 2xx response, the cell shows the new state.
4. On error, the cell reverts and an inline error appears.

The lack of explicit save is intentional — teachers shouldn't need to remember to push a button at the end of the day.

## Per-child attendance history

The weekly grid is the wide view. The narrow view is on the child's detail page, which lists every attendance record for that child in date order. Useful for parent reports ("how many days has Max been absent this term?") and for spotting patterns.

## Reporting and limits

Daily summary endpoints aggregate the per-child records; the dashboard daily summary uses these. The data is exact when every child has a record for the day; it under-counts when cells were left blank.

Deliberately not modelled (matches the Berlin Kita convention): half-day attendance, reason codes, drop-off/pick-up times. Every create/update/delete writes an audit-log entry filterable by `attendance_*` actions.
