---
title: Warum Geldbeträge als Cents gespeichert werden
weight: 7
---

Jeder Geldbetrag in KitaManager wird als **ganzzahlige Anzahl von Cents** gespeichert — `int` in Go, `INTEGER` in Postgres, `number` in TypeScript. Die Konvention gilt überall: Fördersätze, Gehälter, Haushalts-Einträge, die API-Request- und Response-Formen.

Diese Seite erklärt, warum, und was an der Grenze zu tun ist, an der Menschen Euro sehen.

## Das Problem mit Floats

Floating-Point-Zahlen können dezimale Brüche nicht exakt darstellen: `0.1 + 0.2` ist `0.30000000000000004`. Aufkumulierte Rundungsfehler brechen den Bescheid-Vergleich. Ganzzahlige Cents haben dieses Problem nicht.

## Die Konvention

| EUR | Gespeicherter Wert (Cents) |
|---|---|
| €0,01 | 1 |
| €1,00 | 100 |
| €100,00 | 10000 |
| €1.668,47 | 166847 |
| −€23,00 | −2300 |

Negative Beträge sind gültig — werden für den Eltern-Essensbeitrag (Abzug) und für jeden anderen „dies wird zurückgeschuldet"-Eintrag genutzt.

## Konvertierung an der Grenze

**Eingehend (Mensch → ganzzahlige Cents):**

```go
// Importieren eines YAML, in dem Beträge dezimale Euro sind
func euroToCents(eur float64) int {
    return int(math.Round(eur * 100))
}
```

Das `math.Round` ist tragend — `int(2.395 * 100)` ist `239`, nicht `240`, weil Float-Multiplikation nicht exakt 239,5 produziert.

**Ausgehend (ganzzahlige Cents → Mensch):**

```typescript
// Frontend-Anzeige
function formatCurrency(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', {
    style: 'currency',
    currency: 'EUR',
  });
}
```

In Go für Berichte:

```go
fmt.Sprintf("%.2f €", float64(cents)/100)
```

## Wann trotzdem Floats nutzen

Fast nie. Akzeptable Fälle:

- **Anzeige-/Charting-Bibliotheken** (z. B. plotly), die numerische Y-Achsen-Werte annehmen. An der Grenze konvertieren, die Achse mit einem Euro-Formatierer beschriften.
- **Statistische Aggregationen** (mittleres Gehalt, Perzentil von Förderbeträgen), bei denen das Ergebnis informativ, nicht maßgeblich ist. Auch hier in Cents aggregieren und am Ende konvertieren bevorzugen.

Faustregel: wenn die Zahl jemals in einem Vergleich oder einer Summe auftauchen könnte, in Cents halten.

## Wo die Konvention lebt

Die Cents-als-int-Konvention gilt auf jeder Schicht: Postgres-Spalte, Go-Modell, API-JSON, TypeScript-Typ. Die einzige Ausnahme ist hand-erstelltes Förder-YAML (`payment` ist dezimaler EUR für Lesbarkeit; der Importer konvertiert beim Laden zu Cents — siehe [Förder-YAML-Format](../../reference/data-model/funding-yaml-format/)). Die Konvention ist in `CLAUDE.md` dokumentiert.
