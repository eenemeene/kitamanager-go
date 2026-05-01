---
paths:
  - "cmd/**/*.go"
  - "internal/**/*.go"
  - "tools/**/*.go"
  - "scripts/**/*.go"
---

# Go conventions (backend)

Use modern Go (1.22+) idioms throughout. The backend lives under `cmd/`, `internal/`, `tools/`, and `scripts/` — these rules apply to all of them.

## Required modern patterns

- **`any` over `interface{}`** (Go 1.18+).
- **`slices` package over `sort.Slice`**: `slices.SortFunc` with `cmp.Compare` and `cmp.Or`. Use `slices.Contains`, `slices.Index`, etc. instead of manual loops.
- **`maps` package**: `maps.Keys`, `maps.Values`, `maps.Clone`. Combine with `slices.Collect` to gather iterators into slices.
- **`cmp` package**: `cmp.Compare` in sort funcs, `cmp.Or` for multi-field comparisons.
- **`for range N`** (Go 1.22+): `for range 10` or `for i := range 10` instead of `for i := 0; i < 10; i++`.
- **`strings.Repeat`** instead of loop-built repetitions.
- **`log/slog`** (Go 1.21+) for application logging. Never `log.Printf` or `fmt.Printf` for app logging.

## Examples

```go
// CORRECT — modern Go
slices.SortFunc(items, func(a, b Item) int {
    return cmp.Or(
        cmp.Compare(a.Name, b.Name),
        cmp.Compare(a.ID, b.ID),
    )
})
values := slices.Collect(maps.Values(myMap))
if slices.Contains(roles, targetRole) { ... }
for i := range 10 { ... }

// OUTDATED — don't write new code like this
sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
var values []V; for _, v := range myMap { values = append(values, v) }
for _, r := range roles { if r == targetRole { ... } }
for i := 0; i < 10; i++ { ... }
```
