---
title: Kinder zwischen Bereichen verschieben
weight: 3
---

Sie wollen ein Kind von einem Bereich in einen anderen verschieben (z. B. weil es aus der Nest-Gruppe herausgewachsen ist).

## Schneller Weg: Drag-and-Drop

{{< screenshot src="/images/screenshots/sections.png" alt="Bereiche-Ansicht mit einer Spalte pro Bereich" caption="Jeder Bereich ist eine Spalte. Die Karte des Kindes in die Zielspalte ziehen." >}}

1. Klicken Sie in der Seitenleiste auf **Bereiche**.
2. Jeder Bereich ist eine Spalte. Greifen Sie die Karte des Kindes und ziehen Sie sie in die Zielspalte.
3. Loslassen. KitaManager beendet den laufenden Vertrag mit Bis-Datum **gestern** und legt einen neuen Vertrag im Zielbereich mit Beginn **heute** an. Die Historie bleibt damit erhalten.

Passt das Alter des Kindes nicht in den Zielbereich, verschiebt KitaManager es trotzdem, zeigt aber eine Warnung.

### Zwei Ausnahmen

- **Der Vertrag beginnt heute oder später.** Dann wird er direkt geändert, ohne neuen Vertrag — es gibt noch keine Vergangenheit zu bewahren.
- **Der Vertrag ist schon beendet** (Bis-Datum in der Vergangenheit). Ziehen ist dann nicht der richtige Weg: Es gibt keinen laufenden Vertrag, der fortgeschrieben werden könnte. Legen Sie über **Neuer Vertrag** einen Vertrag im Zielbereich an. Soll ein *falsch erfasster* Bereich in der Vergangenheit berichtigt werden, bearbeiten Sie den betreffenden Vertrag in der Vertragshistorie — das ändert ihn an Ort und Stelle und schreibt alte und neue Werte ins Protokoll.

## Manueller Weg: für ein bestimmtes Wechseldatum

Wenn der Wechsel nicht heute gilt (nicht „heute“), sondern zu einem geplanten Datum:

1. Öffnen Sie das Kind über die **Kinder**-Liste und klicken Sie auf das **Verlaufs**-Symbol, um die Vertragshistorie zu öffnen.
2. Suchen Sie den **aktiven** Vertrag, klicken Sie auf den **Stift** und setzen Sie **Bis** auf den Tag vor dem Wechsel. **Speichern**.
3. Klicken Sie auf **Neuer Vertrag**, setzen Sie **Von** auf das Wechseldatum und wählen Sie den neuen **Bereich**. **Speichern**.

Liegt das Wechseldatum in der Vergangenheit, funktioniert nur dieser Weg — siehe den Hinweis zu rückwirkenden Änderungen in [Betreuungsvertrag eines Kindes aktualisieren](../update-child-contract/).

## Hinweise

- Drag-and-Drop ist das richtige Werkzeug für „dieses Kind wechselt jetzt in die nächste Gruppe“. Der manuelle Weg ist für vorausschauende Planung („Max wechselt am 1. August in die Gruppe Große“).
- Das Dashboard-Widget **Kinder über Altersgrenze des Bereichs** zeigt Kinder, die das Höchstalter ihres Bereichs überschritten haben — Verschieben ist die Lösung.
- Für das Bereichs-Altersmodell siehe [Was der Personalschlüssel bedeutet](../../../explanation/what-the-staffing-key-means/).
