---
title: Contract supplements
weight: 2
---

The supplements you can attach to a child's care contract. Each is a `key` / `value` pair on the contract's properties JSON. Each maps to a row in the funding YAML with a payment amount and a staffing requirement.

For human definitions, see the [Glossary](../../glossary/). For where they fit in the funding calculation, see [How contract properties determine funding](../../../explanation/how-contract-properties-determine-funding/).

## Supplements (Berlin)

| UI label | Contract `key` | Contract `value` | YAML location | What it adds |
|---|---|---|---|---|
| NdH | `ndh` | `ndh` | per-age-band entry | small per-child funding supplement + extra staffing FTE |
| QM/MSS | `qm/mss` | `qm/mss` | per-age-band entry | per-child supplement (only if Kita is in a QM/MSS area) |
| Integration A | `integration` | `integration a` | per-age-band entry | larger per-child supplement + significantly higher FTE requirement |
| Integration B | `integration` | `integration b` | per-age-band entry | even larger supplement + higher FTE |

## Care types (also a contract property, not a supplement)

| UI label | `key` | `value` |
|---|---|---|
| Halbtag (≤5h) | `care_type` | `halbtag` |
| Teilzeit (≤7h) | `care_type` | `teilzeit` |
| Ganztag (≤9h) | `care_type` | `ganztag` |
| Ganztag erweitert (>9h) | `care_type` | `ganztag erweitert` |

## Universal deductions

| UI label | `key` | `value` | Notes |
|---|---|---|---|
| Parent meal contribution | `parent` | `meals` | applied to every care contract; currently −€23/month |

## Reading from the funding YAML

The exact `payment` and `requirement` values depend on the active period in `configs/government-fundings/berlin.yaml`. See [Government funding YAML format](../funding-yaml-format/) for the file shape.
