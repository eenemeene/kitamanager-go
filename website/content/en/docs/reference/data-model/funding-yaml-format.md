---
title: Government funding YAML format
weight: 1
---

The funding YAML format describes a state's per-Kita funding rates. The Berlin reference is shipped at [`configs/government-fundings/berlin.yaml`](https://github.com/eenemeene/kitamanager-go/blob/main/configs/government-fundings/berlin.yaml). Same format is consumed by `POST /api/v1/government-funding-rates/import` and by the `GOVERNMENT_FUNDING_SEED_PATH` env var on first start.

For *how to* update the rates, see the operator how-to [Update government funding rates](../../../how-to/operate/update-government-funding-rates/).

## Top-level shape

A YAML file is a list of **funding configurations**. Each configuration represents a time-bounded rate table and contains `entries` (also called periods in the API).

```yaml
- to: ''                       # empty = open-ended
  from: '2026-08-01'           # ISO date
  full_time_weekly_hours: 39   # what counts as full-time
  comment: 'Anlage 1a Nr. XXXIX - Aufschlag für Praxisunterstützungssystem (45€ pro Kind/Jahr) inklusive'
  entries:
    - age: [0, 8]              # min/max age in years
      properties: [...]
    - age: [0, 1]
      properties: [...]
    # ... more age bands
```

## Entry properties

Each entry within a configuration is a list of **property** rates that apply to children whose age falls in the entry's range. A property has a `key`, `value`, `label`, `payment` (in **EUR as a decimal**), and `requirement` (FTE staffing requirement per child for that property).

```yaml
- key: care_type
  value: ganztag
  label: Full-Time (up to 9h)
  payment: 2494.91         # EUR; the importer converts to cents internally
  requirement: 0.355       # 0.355 FTE staff hours per child
```

### Care types

| `key` | `value` | Meaning |
|---|---|---|
| `care_type` | `ganztag erweitert` | Extended full-time (>9h) |
| `care_type` | `ganztag` | Full-time (≤9h) |
| `care_type` | `teilzeit` | Part-time (≤7h) |
| `care_type` | `halbtag` | Half-day (≤5h) |

### Supplements (Zuschläge)

| `key` | `value` | `label` | Meaning |
|---|---|---|---|
| `ndh` | `ndh` | NdH | nichtdeutsche Herkunftssprache — see [How funding works](../../../explanation/how-funding-works-in-berlin/) |
| `qm/mss` | `qm/mss` | QM/MSS | Quartiersmanagement / Monitoring Soziale Stadtentwicklung |
| `integration` | `integration a` | Integration A | Integrationsstatus A under Eingliederungshilfe |
| `integration` | `integration b` | Integration B | Integrationsstatus B (higher support need) |

### Universal deductions

`apply_to_all_contracts: true` makes a property apply to every contract regardless of other properties. Used for the parent meal contribution:

```yaml
- key: parent
  value: meals
  label: Parent Value Meal
  payment: -23.0           # EUR (deduction)
  requirement: 0
  apply_to_all_contracts: true
```

## Money in YAML vs. money inside KitaManager

The funding YAML uses **decimal EUR** for `payment` so a hand-edited file is readable. On import (`POST /api/v1/government-funding-rates/import` and the `GOVERNMENT_FUNDING_SEED_PATH` startup loader), the value is converted to integer cents and stored as such — `int(math.Round(eur * 100))`. Every internal calculation, every API response, and every database column is in **cents**. Round-trip exporting back to YAML re-emits decimal EUR.

For why the storage layer is cents (and the floating-point trap that motivates it), see [Why money is stored as cents](../../../explanation/why-money-is-stored-as-cents/).
