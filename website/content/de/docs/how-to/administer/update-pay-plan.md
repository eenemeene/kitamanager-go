---
title: Entgelttabelle aktualisieren bei TVöD-SuE-Änderungen
weight: 7
---

Die TVöD-SuE-Tabelle ändert sich (typischerweise jährlich). Sie wollen die neuen Gehaltswerte laden, damit zukünftige Verträge und Berichte sie abbilden.

## Schritte

1. Holen Sie das neue Entgelttabellen-YAML — entweder von Ihrer Bezugsstelle, aus dem letzten KitaManager-PR, oder selbst geschrieben nach dem in `make-payplan-yaml` verwendeten Format.
2. Öffnen Sie in der Seitenleiste **Einstellungen** → **Entgelttabellen**.
3. Klicken Sie auf **Importieren**.
4. Wählen Sie die YAML-Datei. KitaManager parst sie und zeigt eine Vorschau.
5. Bestätigen Sie den Import.

Der neue Zeitraum erscheint in der Entgelttabelle. Ab dessen `from`-Datum nutzen Gehaltsberechnungen die neue Tabelle.

## Hinweise

- Die Entgelttabelle ist strukturiert als `Entgelttabelle → Zeiträume → Einträge`. Ein Zeitraum hat ein Von-/Bis-Datum, Wochenstunden und einen Arbeitgeberanteilssatz. Einträge innerhalb eines Zeitraums sind pro-Entgeltgruppe-und-Stufe-Monatsgehälter (in Cents).
- Bestehende Verträge müssen nicht aktualisiert werden — sie referenzieren Entgeltgruppe und Stufe, und die Satz-Suche nutzt den Zeitraum, der den Abrechnungsmonat abdeckt.
- Sie können einen Zeitraum auch manuell in der Oberfläche anlegen (Entgelttabellen → Plan öffnen → Zeitraum hinzufügen). Der YAML-Import ist schneller für die jährliche Komplett-Aktualisierung.
- Gehaltskosten-Berechnungen auf Dashboard, Finanzübersicht und Prognose nutzen sofort nach dem Import die neuen Sätze.
