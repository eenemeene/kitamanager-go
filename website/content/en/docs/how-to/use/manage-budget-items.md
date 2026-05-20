---
title: Manage budget items
weight: 9
---

You want to record an income or expense that isn't automatically computed from contracts (parent fees, rent, garden upkeep, donations) — and keep those numbers current as rent goes up, a category ends, or a one-off cost lands.

## Create a budget item

{{< screenshot src="/images/screenshots/budget-items.png" alt="Budget items list" caption="The budget items list shows each item with its current monthly amount and category." >}}

1. Click **Budget Items** in the sidebar.
2. Click **Create**.
3. Set the **Name** (e.g. "Rent" or "Parent contributions") and pick **Income** or **Expense**.
4. Click **Save**.

## Add entries to the item

Each item holds one or more time-bounded entries. An entry is "this item costs/earns X EUR/month between date A and date B".

{{< screenshot src="/images/screenshots/budget-item-detail.png" alt="Budget item detail with entries" caption="A budget item with several entries — each a time-bounded EUR/month amount." >}}

1. Open the item from the list.
2. Click **Add Entry**.
3. Set:
   - **From** — start date.
   - **To** — end date (leave open for ongoing).
   - **Amount** — euro amount **per month**.
   - **Notes** — optional context (e.g. "incl. Nebenkosten", "annual donation prorated over 12 months").
4. Click **Save**.

{{< screenshot src="/images/screenshots/budget-item-entry-add.png" alt="Add budget item entry dialog" caption="Set the time range, the per-month amount, and optional notes." >}}

The entries feed the **Financial Overview** and the **Forecast** so the cumulative balance and the projection account for them.

## Keep budget items current

Real-world budgets change. Apply the right edit pattern depending on what changed:

### A value changes mid-year (rent goes up, contribution rate is renegotiated)

1. Open the item. The active entry is the one with `To` empty or in the future.
2. Click the **pencil** on that entry. Set **To** to the day before the new amount takes effect (new rent from 1 May → To = 30 April). Save.
3. Click **Add Entry**. Set **From** to the effective date, leave **To** open, enter the new monthly amount. Save.

The detail page now shows two entries back-to-back. Past months keep using the old amount; future months use the new one. The financial overview switches automatically at the boundary.

### A category ends (parent group stops, lease ends)

1. Open the item, click the pencil on the active entry.
2. Set **To** to the last day the amount applies. Save.

The item stays on file with no active entry; historical reports remain correct. Don't delete the item — deletion erases the historical entries too.

### A one-off cost lands (renovation invoice, equipment purchase)

If there isn't already an item for it, create one (category *Expense*) and add a single entry with **From** and **To** both set to the same month (or the month range over which you want to recognise the cost). Notes help your future self remember what it was.

### Fix a wrong amount (input error)

If you typed 200 when you meant 2000, click the pencil and just change the amount on the existing entry — this is correcting history, so editing in place is the right action.

## Notes

- Recurring monthly costs (rent): one entry that spans the whole year.
- One-off costs (renovation): one entry with a short date range.
- Annual lump sums (Spende): split across the relevant months by setting a short date range that approximates the cash flow date.
- Amounts are entered in **euros**, not cents. KitaManager converts to cents internally — see [Why money is stored as cents](../../../explanation/why-money-is-stored-as-cents/).
- The audit log records every entry create / edit / delete with old → new values. Admins can review it via [Review the audit log](../../administer/review-audit-log/).
