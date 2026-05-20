---
title: Update the pay plan when TVöD-SuE rates change
weight: 7
---

The TVöD-SuE pay scale changes (typically annually after each Tarifrunde). You want to load the new salary table so future contracts and reports reflect it. This is an **admin** task per organization — every Kita imports its own pay plan.

## Quick path: import a YAML

If someone already produced a YAML for the new period (a colleague, your provider, or a previous KitaManager release), this is the fastest update.

{{< screenshot src="/images/screenshots/payplan-list.png" alt="Pay plans page" caption="Pay plans for the organization. The Import YAML button is at the top right." >}}

1. Open **Pay Plans** in the sidebar (under the Settings group — admin role required).
2. Click **Import YAML**.
3. Pick the YAML file in the system file picker. The import runs immediately; a toast confirms success.

The new period appears inside the matching pay plan. From its `from` date onwards, salary calculations use the new table.

{{< screenshot src="/images/screenshots/payplan-detail.png" alt="Pay plan detail with periods" caption="A pay plan with multiple validity periods. Each period contains per-grade, per-step monthly amounts." >}}

## Manual path: add a period in the UI

For a small change (one extra step, a single grade correction), or when no YAML is available:

1. Open the pay plan you want to extend from the list.
2. Click **Add Period**. Set **From**, optional **To**, **Weekly hours**, and **Employer contribution rate**. Save.
3. Inside the new period, click **Add Entry** for each grade/step combination. Set **Grade** (e.g. `S 8a`), **Step** (1–6), and **Monthly amount** in euros.
4. Save each entry.

The detail page also has an **Export YAML** button so you can dump the resulting table and share it with peers (or keep it as a backup).

## Notes

- The pay plan is structured as `pay plan → periods → entries`. A period has a from/to date, weekly hours, and an employer contribution rate. Entries within a period are per-grade, per-step monthly salaries.
- Existing contracts don't need updating — they reference grade and step, and the rate lookup uses whichever period covers the bill month.
- Salary cost calculations on the dashboard, financial overview, and forecast all immediately reflect the new rates after import.
- The audit log records every import and every manual change to a period or entry. Admins can review it via [Review the audit log](../review-audit-log/).
- For the salary calculation chain (grade × step × hours × period-rate), see the user-side recipe [Update an employee contract](../../use/update-employee-contract/).
