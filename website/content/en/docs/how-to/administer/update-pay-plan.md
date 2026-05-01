---
title: Update the pay plan when TVöD-SuE rates change
weight: 7
---

The TVöD-SuE pay scale changes (typically annually). You want to load the new salary table so future contracts and reports reflect it.

## Steps

1. Get the new pay-plan YAML — either from your provider, your last KitaManager PR, or write it by hand following the format used by `make-payplan-yaml`.
2. Open **Settings** → **Pay Plans** in the sidebar.
3. Click **Import**.
4. Pick the YAML file. KitaManager parses it and shows a preview.
5. Confirm the import.

The new period appears in the pay plan. From its `from` date onwards, salary calculations use the new table.

## Notes

- The pay plan is structured as `pay plan → periods → entries`. A period has a from/to date, weekly hours, and an employer contribution rate. Entries within a period are per-grade, per-step monthly salaries (in cents).
- Existing contracts don't need updating — they reference grade and step, and the rate lookup uses whichever period covers the bill month.
- You can also create a pay plan period manually in the UI (Pay Plans → open plan → add Period). The YAML import is faster for the full annual update.
- Salary cost calculations on the dashboard, financial overview, and forecast all immediately reflect the new rates after import.
