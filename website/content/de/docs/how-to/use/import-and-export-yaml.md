---
title: YAML-Daten importieren oder exportieren
weight: 14
---

Sie wollen Kinder, Mitarbeiter oder Entgelttabellen massenhaft laden oder exportieren, ohne sie einzeln über die Oberfläche anzulegen.

## Importieren

1. Navigieren Sie zur entsprechenden Seite — **Kinder**, **Mitarbeiter**, **Entgelttabellen** oder **Fördersätze**.
2. Klicken Sie auf **Importieren**.
3. Wählen Sie die YAML-Datei. KitaManager parst sie und zeigt eine Vorschau.
4. Vorschau prüfen, dann bestätigen. Datensätze werden angelegt.

Unterstützte Importe:

| Daten | Format |
|---|---|
| Kinder | YAML |
| Mitarbeiter | YAML |
| Entgelttabellen | YAML |
| Fördersätze | YAML (nur Superadmin — siehe [Berliner Fördersätze aktualisieren](../../operate/update-government-funding-rates/)) |

## Exportieren

1. Navigieren Sie zur entsprechenden Seite.
2. Klicken Sie auf **Exportieren** und wählen Sie das Format.

Unterstützte Exporte:

| Daten | Formate |
|---|---|
| Kinder | Excel, YAML |
| Mitarbeiter | Excel, YAML |
| Entgelttabellen | YAML |

Excel-Exporte öffnen sich in Microsoft Excel, LibreOffice Calc und Google Sheets ohne Konvertierung.

## Hinweise

- **Import legt neue Datensätze an.** Er aktualisiert oder mergt nicht mit bestehenden — gleiche Gutscheinnummern würden zwei Kinder erzeugen. Für Aktualisierungen jeden Datensatz in der Oberfläche oder per API bearbeiten.
- Für die YAML-Form von Förder-Dateien siehe [Förder-YAML-Format](../../../reference/data-model/funding-yaml-format/).
- Für Automatisierung (CI-Exporte, Batch-Loads) die API direkt aufrufen — siehe [API-Referenz](../../../reference/api/).
