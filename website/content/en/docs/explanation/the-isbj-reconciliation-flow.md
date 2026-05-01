---
title: The ISBJ reconciliation flow
weight: 6
---

When you upload an ISBJ Excel via the Funding Bills page, KitaManager runs a multi-stage pipeline. This page describes each stage so you can debug a bill that won't import or a comparison that produces unexpected results.

## Stage 1 — Parse

The Excel file is read with `internal/isbj/parse.go`. The parser:

1. Locates the worksheet that holds the per-child detail rows (sheet names follow a stable Senate convention).
2. Reads each row, normalising column names against an internal map.
3. Extracts: child surname/given-name, voucher number, billed amounts per supplement, K/A markers (corrections — see below).

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

Real ISBJ bills carry "K" (Korrektur) and "A" (Aufhebung) markers on rows that retroactively correct or cancel a previous month's billing. KitaManager's parser **currently ignores these markers**, treating each row as standalone for the bill month. This causes a known billing-comparison drift when the Senate corrects a prior month's amounts: the corrected amount is recorded against the month the bill was *issued in*, not the month it *applies to*, so two months show offsetting deltas instead of one matching row. Tracked as a known limitation; the workaround when triaging is to ignore offsetting deltas across consecutive months.

## Where to look when something goes wrong

| Symptom | Likely cause | Where to look |
|---|---|---|
| "File could not be parsed" | Excel layout drift or wrong file type | `internal/isbj/parse.go` column map |
| "No bills found for this month" | Upload created a different period than expected | period dates in `government_funding_bill_periods` |
| Many "missing from bill" entries | Voucher numbers don't match | child detail pages, [Investigate a discrepancy](../../how-to/use/investigate-a-bill-discrepancy/) |
| Many "extra in bill" entries | Children left but Senate hasn't updated | Bezirks-Jugendamt |
| Persistent total drift across many children | Funding rates outdated | [Update government funding rates](../../how-to/operate/update-government-funding-rates/) |
| Total matches but property breakdown differs | Supplement out of sync | child contract supplements vs paper voucher |
