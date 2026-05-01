---
title: Why money is stored as cents
weight: 7
---

Every monetary value in KitaManager is stored as an **integer count of cents** — `int` in Go, `INTEGER` in Postgres, `number` in TypeScript. The convention applies everywhere: funding rates, salary amounts, budget entries, the API request and response shapes.

This page explains why, and what to do at the boundary where humans see euros.

## The problem with floats

Floating-point numbers can't represent decimal fractions exactly. The classic demo:

```go
fmt.Println(0.1 + 0.2)
// Output: 0.30000000000000004
```

For a system that compares thousands of monthly bills against thousands of calculated amounts, even tiny rounding errors compound into systematic drift. A €0.01 floating-point miscompare on every one of 120 children every month is €14.40 of phantom drift per year per organisation. Multiply across many Kitas and many years and the trust in the comparison disappears.

Integer cents have no such problem: `10 + 20 = 30`, exactly, every time.

## The convention

| EUR | Stored value (cents) |
|---|---|
| €0.01 | 1 |
| €1.00 | 100 |
| €100.00 | 10000 |
| €1,668.47 | 166847 |
| −€23.00 | −2300 |

Negative amounts are valid — used for the parent meal contribution (a deduction) and for any other "this is owed back" entry.

## Conversion at the boundary

**Inbound (human → integer cents):**

```go
// Importing a YAML where amounts are decimal euros
func euroToCents(eur float64) int {
    return int(math.Round(eur * 100))
}
```

The `math.Round` is load-bearing — `int(2.395 * 100)` is `239`, not `240`, because float multiplication doesn't produce exactly 239.5.

**Outbound (integer cents → human):**

```typescript
// Frontend display
function formatCurrency(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', {
    style: 'currency',
    currency: 'EUR',
  });
}
```

In Go for reports:

```go
fmt.Sprintf("%.2f €", float64(cents)/100)
```

## When to use floats anyway

Almost never. Acceptable cases:

- **Display/charting libraries** (e.g. plotly) that take numeric Y-axis values. Convert at the boundary, label the axis with a euro formatter.
- **Statistical aggregations** (mean salary, percentile of funding amounts) where the result is informational, not authoritative. Even here, prefer aggregating in cents and converting at the end.

The rule of thumb: if the number could ever appear in a comparison or a sum, keep it in cents.

## Where the convention lives

- Postgres columns: `INTEGER NOT NULL`.
- Go models: `int` (not `int64`, which would suggest amounts large enough to overflow `int32` — KitaManager's domain doesn't reach that range).
- API JSON: integer `payment` field, never a string.
- TypeScript types: `number` (auto-generated from OpenAPI).
- Funding YAML: integer `payment` (see [Government funding YAML format](../../reference/data-model/funding-yaml-format/)).
- The convention is documented in `CLAUDE.md` as a cross-cutting rule that applies to every change.
