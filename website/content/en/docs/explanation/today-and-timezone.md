---
title: How "today" and timezone work
weight: 6
---

KitaManager makes lots of "is this contract active today?" decisions: which children appear on the attendance grid this week, which contracts the staffing-hours chart counts for the current month, whether a Stufenaufstieg is "due now". The answer to "what is today?" must match what the user sees on their wall clock — not the server's UTC clock.

## The rule

Every "today's calendar date" decision goes through `models.Today()`. It returns the UTC midnight of the current calendar date in the **application's** timezone — `Europe/Berlin` by default, override via the `KITAMANAGER_TIMEZONE` env var.

Why this matters: if you ask "is this contract active today?" at 23:30 Berlin local time on Sept 30, the server's UTC clock thinks it's 22:30 on Sept 30 — same answer. But at 01:00 Berlin local time on Oct 1, the server's UTC clock says 23:00 on Sept 30 — the previous day. A Berlin user would expect "today is Oct 1"; a naive UTC truncation would say Sept 30, and a contract starting Oct 1 would appear inactive for an hour every night.

## Where it shows up

- The attendance grid lists children whose contracts are active "today" in Berlin.
- The dashboard's KPI tiles use the current Berlin month.
- The Stufenaufstieg widget uses Berlin "today" to decide which steps are due.
- The auto-derived attendance date when you tap a cell is Berlin "today".
- The future-birthdate guard ("you can't enrol a child born in the future") uses Berlin "today".

`time.Now()` is still the right call for *instant* timestamps — audit log entries, JWT issued-at, MFA expiry, attendance check-in/out — because those want the precise moment, not a calendar day.

## Changing the timezone

Set `KITAMANAGER_TIMEZONE=Europe/Vienna` (or any IANA zone name) and restart the API. The container ships embedded tzdata so any zone resolves regardless of base image.

If you change the timezone on a running system with existing data, only *future* "today" calculations move. Already-recorded attendance dates and audit timestamps don't shift — they were the right calendar date for the previous timezone.

## Pin "today" in tests

For developer reference: `models.SetNow(instant)` overrides the time source for the duration of a test. The seam exists so date-rollover bugs are reproducible in CI — without it they only manifest when the runner happens to cross the timezone boundary at the right moment.
