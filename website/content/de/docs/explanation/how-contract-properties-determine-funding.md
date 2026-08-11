---
title: Wie Vertragseigenschaften die Förderung bestimmen
weight: 2
---

Der monatliche Förderbetrag eines Kindes in KitaManager ist das Ergebnis einer deterministischen Suche in der Förder-Konfiguration. Diese Seite erklärt die Berechnungskette, damit Sie vorhersagen können, was eine Änderung eines Vertragsfeldes mit dem berechneten Betrag macht — und damit Sie eine Abweichung von Grund auf debuggen können.

## Die Eingaben

Drei Dinge aus dem Betreuungsvertrag treiben die Berechnung:

1. **Das Geburtsdatum des Kindes** — wird genutzt, um sein Alter zum Abrechnungsmonat zu berechnen.
2. **Die `care_type`** — Halbtag, Teilzeit, Ganztag oder Ganztag erweitert.
3. **Die Menge der Zuschläge** — NdH, QM/MSS, Integration A, Integration B (null oder mehr).

Die Berechnung liest außerdem:

- **Den Abrechnungsmonat** — Fördersätze ändern sich über die Zeit, daher ist die Suche an ein Datum verankert.
- **Die Förder-Konfiguration** für das Bundesland der Organisation — derzeit immer `berlin` für die mitgelieferten Daten.

## Die Berechnungskette

Für jedes (Kind, Monat)-Paar:

1. **Aktiven Förderzeitraum finden.** Das Förder-YAML ist eine Liste von Konfigurationen, jede mit `from`-Datum und optionalem `to`-Datum. Die Konfiguration wählen, deren Zeitraum den Abrechnungsmonat abdeckt.
2. **Alter des Kindes in Jahren berechnen** zu Beginn des Abrechnungsmonats. Das Alter bestimmt, welcher Eintrag innerhalb der Konfiguration gilt.
3. **Den Eintrag wählen**, dessen `age: [min, max]`-Bereich das Alter des Kindes abdeckt.
4. **Den Grundsatz innerhalb dieses Eintrags suchen.** Der Grundsatz ist die Eigenschaft, deren `key` `care_type` ist und deren `value` zum `care_type` des Vertrages passt.
5. **Jeden Zuschlagsbetrag addieren.** Für jeden auf dem Vertrag gesetzten Zuschlag (`ndh`, `qm/mss`, `integration` mit Wert `integration a` oder `integration b`) die passende Eigenschaft im selben Eintrag suchen und ihren `payment` zur laufenden Summe addieren.
6. **Die universellen Abzugs-Eigenschaften addieren.** Eigenschaften mit `apply_to_all_contracts: true` (derzeit der Eltern-Essensbeitrag bei −€23) werden bei jedem Vertrag unabhängig von dessen anderen Eigenschaften addiert.

Das Ergebnis, in Cents, ist das, was KitaManager als „berechnete Förderung“ für dieses Kind in diesem Monat anzeigt.

## Ein durchgerechnetes Beispiel

Nehmen wir ein 2-jähriges Kind mit `ganztag`-Vertrag und gesetztem NdH, im Oktober 2026. Suche gegen `configs/government-fundings/berlin.yaml`:

1. Aktive Konfiguration: die mit `from: 2026-08-01`, kein `to`.
2. Alter des Kindes: 2.
3. Passender Eintrag: `age: [2, 3]`.
4. Grundsatz: `care_type: ganztag` in diesem Eintrag → z. B. `payment: 2240.12` (€2.240,12 im YAML, gespeichert als 224 012 Cents).
5. Zuschläge: NdH → `payment: 93.51` (9 351 Cents). Plus der universelle Eltern-Essensbeitrag: `payment: -23.0` (−2 300 Cents).
6. Gesamt: 224 012 + 9 351 − 2 300 = **231 063 Cents = €2.310,63**.

(Die exakten Zahlen verschieben sich jedes Jahr — was zählt, ist die Kette.)

## Warum die Kette in der Praxis wichtig ist

Die meisten „die Abrechnung passt nicht zu KitaManager“-Fälle führen auf eine der Eingaben zurück:

- **Falsche `care_type` auf dem Vertrag** → falscher Grundsatz. Die Abweichung kann hunderte Euro pro Kind und Monat ausmachen.
- **Fehlender oder veralteter Zuschlag** (NdH, QM/MSS, Integration) → der Zuschlagsbetrag fehlt stillschweigend. Kleinere pro-Kind-Wirkung (€60–€350), aber sehr häufig.
- **Veraltete Förder-Konfiguration** → jedes Kind um den gleichen Delta daneben. Sieht im Bescheid-Vergleich wie systematische Drift über viele Kinder aus.
- **Falsches Geburtsdatum** (selten, aber möglich) → fällt in einen anderen alters-basierten Eintrag, Grundsatz ist falsch.

Für das Rezept siehe [Abweichung in einer Abrechnung untersuchen](../../how-to/use/investigate-a-bill-discrepancy/). Für das YAML-Format siehe [Förder-YAML-Format](../../reference/data-model/funding-yaml-format/).

## Verwandte Erklärungen

- [Wie die Förderung in Berlin funktioniert](../how-funding-works-in-berlin/) — der institutionelle Kontext (wer setzt die Sätze, wer stellt die Gutscheine aus, wer betreibt ISBJ).
- [Der ISBJ-Abgleich](../the-isbj-reconciliation-flow/) — was KitaManager mit einer hochgeladenen Excel macht.
