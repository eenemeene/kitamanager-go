---
title: Berliner Fördersätze aktualisieren
weight: 3
---

Die Berliner Senatsverwaltung veröffentlicht neue Kostenblatt-Beträge (typischerweise einmal jährlich, am 1. August). Sie wollen die neuen Sätze laden, damit KitaManagers Berechnungen zu den neuen ISBJ-Bescheiden passen. Das ist eine **Superadmin-only**-Operation.

## Schritte

### Variante 1: Offizielles YAML importieren

1. Holen Sie das aktualisierte YAML — typischerweise wird eines im Projektrepository als `configs/government-fundings/berlin.yaml` ausgeliefert.
2. Melden Sie sich als Superadmin an.
3. Öffnen Sie **Einstellungen** → **Fördersätze**.
4. Klicken Sie auf **Importieren**.
5. Wählen Sie die YAML-Datei. Die Vorschau zeigt, was angelegt wird.
6. Bestätigen.

### Variante 2: Eine Eigenschaft in der Oberfläche bearbeiten

Für eine einzelne Korrektur:

1. Förder-Konfiguration → relevanten Zeitraum → relevanten Eintrag öffnen.
2. Eigenschaft bearbeiten (`payment` ist in Cents — €2.494,91 → `249491`).
3. **Speichern** klicken.

## Hinweise

- Fördersätze sind **global**, nicht pro Organisation. Eine Änderung wirkt auf jede Kita im System.
- Das Format ist dokumentiert unter [Förder-YAML-Format](../../../reference/data-model/funding-yaml-format/).
- Im YAML ist `payment` **dezimaler EUR-Wert** (z. B. `2494.91`). Beim Import konvertiert KitaManager zu Cents (`249491`) — das ist die intern gespeicherte Form.
- Nach dem Update: der nächste Bescheid-Vergleich nutzt die neuen Sätze ab dem Bescheid-Monat. Vergangene Bescheide bleiben in der Speicherung unverändert, aber ihr erneuter Vergleich würde nun andere Zahlen erzeugen.
- Audit-Log dokumentiert jede Eigenschafts-Bearbeitung mit Alt → Neu.
