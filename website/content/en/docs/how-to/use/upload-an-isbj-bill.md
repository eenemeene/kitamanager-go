---
title: Upload a funding bill
weight: 6
---

Every month the Senate sends an Excel file with the funding amounts for your children — the **Senatsabrechnung**, also called the *ISBJ Bescheid*. You upload it so KitaManager can compare it against its own calculation.

## Steps

{{< screenshot src="/images/screenshots/government-funding-bills.png" alt="Government Funding Bills page with the Excel file picker" caption="The **Select ISBJ Excel file (.xlsx)** panel sits at the top of the page. The **Upload** button only becomes active once a file is chosen." >}}

1. Click **Funding Bills** in the sidebar.
2. In the **Select ISBJ Excel file (.xlsx)** panel, pick the file from your computer.
3. Click **Upload**. The button stays greyed out until a file is chosen.
4. The bill appears in the table below, grouped by Kita year.
5. The bar above the table shows straight away: how many bills match, how many have differences, and the total difference in euros.
6. Click the **eye icon** in the **Actions** column to open the child-by-child comparison.

## What KitaManager does with the file

KitaManager reads the Excel file and matches every row to a care contract by its Gutscheinnummer. For each matched child it compares the amounts.

Anything it cannot match appears in one of two groups:

- **Missing from bill** — KitaManager knows the child, the bill does not list them.
- **Extra in bill** — the bill lists a child KitaManager does not know.

The full sequence is described in [The ISBJ reconciliation flow](../../../explanation/the-isbj-reconciliation-flow/).

## Notes

- Upload the same month's bill again and it replaces the previous one: the old rows are deleted and the new ones inserted. The audit log records the replacement.
- Each Kita uploads its own bill — a bill always belongs to exactly one organisation.
- Discrepancies don't fix themselves. The next step is [Investigate a bill discrepancy](../investigate-a-bill-discrepancy/).
