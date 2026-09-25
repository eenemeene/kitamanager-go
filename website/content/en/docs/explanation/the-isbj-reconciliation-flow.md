---
title: The ISBJ reconciliation flow
weight: 6
---

When you upload an ISBJ Excel via the Funding Bills page, KitaManager runs a multi-stage pipeline. This page describes each stage so you can debug a bill that won't import or a comparison that produces unexpected results.

## Stage 1 — Parse

The Excel file is read with `internal/isbj/parse.go`. The parser:

1. Locates the worksheet that holds the per-child detail rows (sheet names follow a stable Senate convention).
2. Reads each row, normalising column names against an internal map.
3. Extracts: child surname/given-name, voucher number, billed amounts per supplement, and each row's type and applicable month (corrections — see below).

Parse errors are surfaced inline. Common causes:

- The Excel layout has changed in a Senate update — the parser's column map needs updating.
- The file isn't actually an ISBJ Bescheid (e.g. an unrelated XLSX was uploaded).
- The worksheet is empty.

## Stage 2 — Persist

Successfully parsed rows are stored in two places:

- A `government_funding_bill_period` for the upload as a whole (covers the date range, references the file).
- One `government_funding_bill_entry` per row.

Re-uploading the same period replaces the previous one — the previous bill's entries are deleted before the new ones are inserted. Audit log records both events.

## Stage 3 — Match

For each entry, KitaManager looks for a child in the organisation with that voucher number. The match is on `voucher_number` only — names are not used for matching, because Senate-level renames and capitalisation differences are common.

- **Match found** → the entry pairs with the child. KitaManager looks at the child's active contract for the bill month.
- **No match** → the entry is flagged "extra in bill".

Children that have an active contract for the bill month but whose voucher number doesn't appear in the bill at all are flagged "missing from bill".

## Stage 4 — Compare

For each matched (entry, child contract) pair, KitaManager calculates what *it* would expect to bill for the contract month using the funding configuration (see [How contract properties determine funding](../how-contract-properties-determine-funding/)). It then compares against the entry's billed amount.

- **Match** (within rounding) → the row is green.
- **Different** → the row is red, with the delta shown.

The comparison is per-property: not just total amount, but per-supplement amount. So a child where the total matches but the breakdown differs (e.g. one side has NdH, the other has Integration A) is flagged "different" with property-level detail.

## K/A markers (corrections)

Real ISBJ bills carry two values per row in the "Monat/ Typ" column: the **Typ** — "A" for *Abrechnung* (the regular row) or "K" for *Korrektur* (a correction) — and the **Monat**, the month the row applies to.

Those two are not the same as the bill's own month. A bill for April routinely looks like this:

| Typ | Monat | Amount |
|---|---|---|
| K | 01.25 | €1.02 |
| K | 02.25 | €1.02 |
| K | 03.25 | €1.02 |
| A | 04.25 | €946.50 |

The Senate is saying: here is April — and the rate we paid you in January, February and March was €1.02/month short each. Retroactive corrections arise when the Kostenblatt is republished after the fact, an Integrationsstatus is approved with an effective date in the past, or an NdH flag is corrected later.

KitaManager reads both values and files each row against the month it applies to. Two views therefore answer two different questions, and both are right:

- **Financials / balance** counts by **arrival month** — what actually landed in that month, corrections included. That is the cash question.
- **Bill comparison** counts by **applicable month** — was this month funded correctly. The three corrections above count towards January, February and March there, not April.

{{< callout type="info" >}}
For bills imported before this behaviour existed, the applicable month was not stored. Those rows still count towards their arrival month, so historical figures do not shift retroactively. To get the attribution for older bills as well, re-import the files in question.
{{< /callout >}}

When a correction applies to a month for which no bill was imported at all, no difference can be formed for that month — there is no calculated figure to compare against. The amount is not dropped: it is reported separately beneath the "Correction" column in the Kita year row.

For the operational triage matrix (which symptom maps to which fix), see [Investigate a bill discrepancy](../../how-to/use/investigate-a-bill-discrepancy/).
