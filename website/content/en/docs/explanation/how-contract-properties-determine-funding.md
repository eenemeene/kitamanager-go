---
title: How contract properties determine funding
weight: 2
---

A child's monthly funding amount in KitaManager is the result of a deterministic lookup against the funding configuration. This page explains the lookup chain so you can predict what changing a contract field will do to the calculated amount, and so you can debug a mismatch from first principles.

## The inputs

Three things from the child contract drive the calculation:

1. **The child's birthdate** — used to compute their age at the bill month.
2. **The `care_type`** — Halbtag, Teilzeit, Ganztag, or Ganztag erweitert.
3. **The set of supplements** — NdH, QM/MSS, Integration A, Integration B (zero or more).

The calculation also reads:

- **The bill month** — funding rates change over time, so the lookup is anchored to a date.
- **The funding configuration** for the organisation's state — currently always `berlin` for the shipped data.

## The lookup chain

For each (child, month) pair:

1. **Find the active funding period.** The funding YAML is a list of configurations, each with a `from` and optional `to` date. Pick the configuration whose date range covers the bill month.
2. **Compute the child's age in years** at the start of the bill month. The age determines which entry within the configuration applies.
3. **Pick the entry** whose `age: [min, max]` range covers the child's age.
4. **Look up the base rate** within that entry. The base rate is the property whose `key` is `care_type` and whose `value` matches the contract's care_type.
5. **Add each supplement amount.** For every supplement set on the contract (`ndh`, `qm/mss`, `integration` with value `integration a` or `integration b`), look up the matching property in the same entry and add its `payment` to the running total.
6. **Add the universal-deduction properties.** Properties marked `apply_to_all_contracts: true` (currently the parent meal contribution at −€23) are added to every contract regardless of its other properties.

The result, in cents, is what KitaManager displays as "calculated funding" for that child for that month.

## A worked example

Take a 2-year-old child on a `ganztag` contract with NdH set, in October 2026. Look up against `configs/government-fundings/berlin.yaml` (Berlin):

1. Active configuration: the one with `from: 2026-08-01`, no `to`.
2. Child age: 2.
3. Matching entry: `age: [0, 1]` — wait, the child is 2 so this doesn't match. Try the next: `age: [2, 3]`.
4. Base rate: `care_type: ganztag` in that entry → e.g. `payment: 224012` (2,240.12 €).
5. Supplements: NdH → `payment: 9351` (93.51 €). Plus the parent-meal universal: `−2300` (−23.00 €).
6. Total: 224012 + 9351 − 2300 = **231063 cents = 2,310.63 €**.

(The exact numbers shift each year — what matters is the chain.)

## Why the chain matters in practice

Most "the bill doesn't match KitaManager" cases trace to one of the inputs:

- **Wrong `care_type` on the contract** → wrong base rate. The mismatch can be hundreds of euros per child per month.
- **Missing or stale supplement** (NdH, QM/MSS, Integration) → the supplement amount is silently absent. Smaller per-child impact (€60–€350) but very common.
- **Stale funding configuration** → every child off by the same delta. You'll see it as a systematic drift across the bill comparison.
- **Wrong birthdate** (rare but possible) → falls into a different age-based entry, base rate is wrong.

For the recipe, see [Investigate a bill discrepancy](../../how-to/use/investigate-a-bill-discrepancy/). For the YAML format, see [Government funding YAML format](../../reference/data-model/funding-yaml-format/).

## Related explanations

- [How funding works in Berlin](../how-funding-works-in-berlin/) — the institutional context (who sets the rates, who issues the vouchers, who runs ISBJ).
- [The ISBJ reconciliation flow](../the-isbj-reconciliation-flow/) — what KitaManager does with an uploaded Excel.
