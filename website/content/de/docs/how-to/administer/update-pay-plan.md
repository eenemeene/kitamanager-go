---
title: Entgelttabelle aktualisieren bei TVöD-SuE-Änderungen
weight: 7
---

Die TVöD-SuE-Tabelle ändert sich (typischerweise jährlich nach jeder Tarifrunde). Sie wollen die neuen Gehaltswerte laden, damit zukünftige Verträge und Berichte sie abbilden. Das ist eine **Admin**-Aufgabe pro Organisation — jede Kita importiert ihre eigene Entgelttabelle.

## Schnellweg: YAML importieren

Wenn bereits ein YAML für den neuen Zeitraum existiert (Kolleg:in, Bezugsstelle oder ein KitaManager-Release), ist das der schnellste Weg.

{{< screenshot src="/images/screenshots/payplan-list.png" alt="Entgelttabellen-Übersicht" caption="Entgelttabellen der Organisation. Der Button YAML importieren ist oben rechts." >}}

1. **Entgelttabellen** in der Seitenleiste öffnen (unter der Einstellungen-Gruppe — Admin-Rolle erforderlich).
2. Auf **YAML importieren** klicken.
3. Die YAML-Datei im System-Dateidialog auswählen. Der Import läuft sofort; ein Toast bestätigt den Erfolg.

Der neue Zeitraum erscheint in der passenden Entgelttabelle. Ab dessen `from`-Datum nutzen Gehaltsberechnungen die neue Tabelle.

{{< screenshot src="/images/screenshots/payplan-detail.png" alt="Entgelttabelle mit Zeiträumen" caption="Eine Entgelttabelle mit mehreren Gültigkeitszeiträumen. Jeder Zeitraum enthält Monatsbeträge pro Entgeltgruppe und Stufe." >}}

## Manueller Weg: Zeitraum in der Oberfläche anlegen

Für eine kleine Änderung (eine zusätzliche Stufe, eine einzelne Korrektur) oder wenn kein YAML vorliegt:

1. Die zu ergänzende Entgelttabelle aus der Liste öffnen.
2. **Zeitraum hinzufügen** klicken. **Von**, optional **Bis**, **Wochenstunden** und **Arbeitgeberanteilssatz** setzen. Speichern.
3. Innerhalb des neuen Zeitraums **Eintrag hinzufügen** für jede Entgeltgruppen-/Stufen-Kombination. **Entgeltgruppe** (z. B. `S 8a`), **Stufe** (1–6) und **Monatsbetrag** in Euro setzen.
4. Jeden Eintrag speichern.

Die Detailseite hat zusätzlich einen Button **YAML exportieren**, mit dem Sie die Tabelle ausgeben und mit anderen teilen können (oder als Backup behalten).

## Hinweise

- Die Entgelttabelle ist strukturiert als `Entgelttabelle → Zeiträume → Einträge`. Ein Zeitraum hat ein Von-/Bis-Datum, Wochenstunden und einen Arbeitgeberanteilssatz. Einträge innerhalb eines Zeitraums sind Monatsgehälter pro Entgeltgruppe und Stufe.
- Bestehende Verträge müssen nicht aktualisiert werden — sie referenzieren Entgeltgruppe und Stufe, und die Satz-Suche nutzt den Zeitraum, der den Abrechnungsmonat abdeckt.
- Gehaltskosten-Berechnungen auf Dashboard, Finanzübersicht und Prognose nutzen sofort nach dem Import die neuen Sätze.
- Das Protokoll erfasst jeden Import und jede manuelle Änderung an Zeitraum oder Eintrag. Admins können es einsehen: [Protokoll prüfen](../review-audit-log/).
- Zur Berechnungskette für Gehälter (Entgeltgruppe × Stufe × Stunden × Zeitraumsatz): [Arbeitsvertrag aktualisieren](../../use/update-employee-contract/).
