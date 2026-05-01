---
title: Why the Kita year runs Aug–Jul
weight: 4
---

KitaManager treats the Kita year as **August 1 → July 31**, not the calendar year. This is the German school-year convention: schools and Kitas run their intake, staffing, and budget cycles over this period.

If you're a developer looking at financial reports and wondering why the cumulative balance "resets" in August: this page explains.

## What follows the Kita year

- **Cumulative balance** on the Financial Overview resets to zero on 1 August.
- **Funding-rate periods** in the Berlin Kostenblatt typically start on 1 August.
- **TVöD-SuE pay-step advancements** for educators often align to the school year boundary.
- **Section assignments** for children traditionally rotate at the school-year transition (the Nest cohort moves up to Nestflüchter, etc.).

## What does *not* follow the Kita year

- The **calendar year** is still the unit for tax reporting, parent fee statements, and many external statistics.
- **Audit log** entries are timestamped against the calendar (UTC instants); filters use calendar dates.
- **Attendance** is recorded per calendar day.
- **Pay plans** can have any from/to range; they don't have to align to the Kita year (though TVöD-SuE updates often do).

## Where Kita-year boundaries appear in the UI

- The Financial Overview chart has alternating shaded bands marking Kita years. Each band is one Aug–Jul period.
- The funding-bill upload page groups bills by Kita year.
- The cumulative-balance chart's deficit markers reset at each August boundary.

## Choosing the right time aggregation

When you need to *compare year-over-year*, use the Kita year. When you need to *report tax-relevant numbers*, use the calendar year. The Statistics page lets you set arbitrary `from` / `to` dates so both views are possible.
