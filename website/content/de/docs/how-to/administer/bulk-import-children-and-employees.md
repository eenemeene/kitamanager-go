---
title: Kinder und Mitarbeiter aus YAML massenimportieren
weight: 11
---

Sie haben eine Liste an Kindern oder Mitarbeitenden zu laden — typischerweise beim ersten Übergang von Tabellen, oder für einen ganz neuen Bereich, der im August startet.

## Schritte

1. In der Seitenleiste auf **Kinder** oder **Mitarbeiter** navigieren.
2. **Importieren** klicken.
3. YAML-Datei wählen. KitaManager liest sie und zeigt eine Vorschau der anzulegenden Datensätze.
4. Sorgfältig prüfen. **Importe legen an, sie mergen nicht** — doppelte Gutscheinnummern oder doppelte (Vorname, Nachname, Geburtsdatum)-Tupel erzeugen Duplikate, die manuell aufzuräumen wären.
5. Bestätigen.

## Einfachster Weg: exportieren, bearbeiten, importieren

Der sauberste Weg, eine YAML zu schreiben, ist **die bestehenden Daten zuerst zu exportieren** (Kinder → Exportieren → YAML, oder Mitarbeiter → Exportieren → YAML), die Struktur zu kopieren und die Werte für die neuen Datensätze zu bearbeiten. Der Exporter emittiert exakt die Form, die der Importer akzeptiert; kein Raten der Feldnamen.

## YAML-Mindestform — Kinder

```yaml
children:
  - first_name: Max
    last_name: Mustermann
    gender: male                          # male | female | diverse
    birthdate: '2024-03-15'
    vouchers:
      - '4711-2026-08-AB'                 # Liste auf Kind-Ebene (nicht am Vertrag)
    contracts:
      - from: '2026-08-01'
        to: null                          # null = offen
        section_name: Nest                # nach Name
        properties:
          care_type: ganztag
          ndh: ndh                        # weglassen, wenn nicht zutreffend
```

## YAML-Mindestform — Mitarbeiter

```yaml
employees:
  - first_name: Anna
    last_name: Muster
    gender: female
    birthdate: '1990-06-21'
    contracts:
      - from: '2026-08-01'
        to: null
        staff_category: qualified         # qualified | supplementary | non_pedagogical
        grade: S8a
        step: 3
        weekly_hours: 39
        payplan_name: TVöD-SuE 2024       # nach Name
        section_name: Nest
```

## Verifizierung nach dem Import

- **Kinder** / **Mitarbeiter** öffnen und 3 zufällige Datensätze stichprobenartig prüfen: stimmen Vertragsdaten, Bereiche und Eigenschaften?
- Dashboard öffnen. **Kinder ohne Gutscheinnummer** sollte leer sein, falls Sie alle `vouchers`-Listen gefüllt haben.
- **Statistiken → Personalstunden** für den Import-Monat öffnen. Benötigte und verfügbare Stunden sollten den neuen Personalstand abbilden.

## Hinweise

- Entgelttabellen und Bereiche werden nach Name aufgelöst. Sie müssen vor dem Import in der Organisation existieren. Bei einer frischen Organisation zuerst Bereiche anlegen und die Entgelttabelle importieren.
- Für das (andere — Superadmin-only) Förder-YAML-Format siehe [Förder-YAML-Format](../../../reference/data-model/funding-yaml-format/) und [Berliner Fördersätze aktualisieren](../../operate/update-government-funding-rates/).
- Für den Export zurück siehe [YAML-Daten importieren oder exportieren](../../use/import-and-export-yaml/).
