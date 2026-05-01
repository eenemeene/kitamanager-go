---
title: What the staffing key means
weight: 3
---

KitaManager's **staffing coverage** number is a one-line summary of "do you have enough staff for the children you have?" This page explains what's actually being computed.

## The two sides of the equation

For any given month, KitaManager computes two numbers:

- **Required hours** — the staff hours the funding configuration says your children need. Each contract property has a `requirement` field (in FTE) — the per-child staffing demand for that property. The state-level requirement totals are summed across all active children for the month.
- **Available hours** — the staff hours your active employee contracts provide. Sum of `weekly_hours` across employees whose contract is active in that month, scaled to a monthly total.

The dashboard's **Staffing Coverage** percent is `(available − required) / required × 100`, capped to a sane sign (negative = understaffed, positive = surplus).

## How the requirement comes from the funding config

Every property in the funding YAML carries a `requirement`:

```yaml
- key: care_type
  value: ganztag
  payment: 249491
  requirement: 0.355   # 0.355 FTE per child for full-time care
- key: ndh
  value: ndh
  payment: 9351
  requirement: 0.017   # additional FTE for NdH children
```

A 1-year-old on `ganztag` with NdH contributes `0.355 + 0.017 = 0.372` FTE to the required side. Multiply by the total weekly hours that count as full-time (`full_time_weekly_hours: 39` for Berlin), then by the number of weeks in the month, and you get the required staff hours for that one child.

Sum across every active child in the month → required hours total.

## How the availability comes from contracts

For each employee:

1. Find every contract that overlaps the bill month.
2. Use the contract's `weekly_hours` × the active days in the month / days-in-month → the contract's contribution.

Sum across every employee → available hours total.

## What the percent actually tells you

| Percent | Meaning |
|---|---|
| `+0%` | Exactly staffed for the funding-config requirement. |
| `+10%` | 10% more available staff than required. Surplus. |
| `−10%` | 10% short. Children whose care depends on the missing hours are not getting the requirement-implied attention. |
| `−40%` or worse | Severe understaffing. The Senate-defined ratio is being violated. |

A persistent surplus is not always good — it means you're paying for staff hours that aren't required by your enrolled children. A persistent deficit means you can't deliver the care the funding rates assume.

## Per-section view

Statistics → Staffing Hours has both an organisation-wide chart and a per-section breakdown. Children's `requirement` is summed by their assigned section's contracts; employees' `available` is summed by their assigned section.

A balanced organisation total can hide a section-level imbalance (one section understaffed, another overstaffed). Check both views.

## Caveats

- **Vacation isn't modelled.** Available hours assume contracts run continuously. Real-world holiday absences aren't deducted.
- **Sickness isn't modelled.** Same reason.
- **Non-pedagogical staff (Hauswirtschaft, Verwaltung) shouldn't count toward `available`** — but if you classify them with a `staff_category` that's listed under the funding YAML as not-pedagogical, the calculation correctly excludes them. Check the per-employee view if numbers look off.
- **The funding `requirement` is the Senate's number, not yours.** If your Kita has a higher pedagogical standard (more staff per child), you'll always show a positive coverage. That's by design — the Senate's number is the floor.
