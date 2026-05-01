---
title: Upload an ISBJ bill
weight: 6
---

You want to upload your monthly ISBJ Bescheid (the Excel file the Senate sends you) so KitaManager can compare it against its own funding calculation.

## Steps

1. Click **Funding Bills** in the sidebar.
2. Click **Upload** and select the Excel file from your computer.
3. The bill appears in the list, grouped by Kita year.
4. The summary bar shows immediately how many children matched, how many had differences, and the total monetary delta.
5. Click the bill row to open the per-child comparison.

## What happens behind the scenes

KitaManager parses the Excel, normalises it into per-child entries, and joins each entry against your child contracts using the Gutscheinnummer. For each match it compares amounts; for each non-match it categorises as **missing from bill** or **extra in bill**. See [The ISBJ reconciliation flow](../../../explanation/the-isbj-reconciliation-flow/) for the full pipeline.

## Notes

- Re-uploading the same month's bill replaces the previous one — the old bill's per-child rows are deleted and the new ones inserted. Audit log records the replacement.
- Bills are scoped to one organisation. Each Kita uploads its own.
- Discrepancies don't fix themselves. Once you've uploaded, [Investigate a bill discrepancy](../investigate-a-bill-discrepancy/) is the natural next step.
