---
title: Wie Anwesenheit modelliert wird
weight: 5
---

Anwesenheit in KitaManager ist tagesgenau, pro Kind. Diese Seite erklärt das Datenmodell, damit das Verhalten der Oberfläche („warum erscheint diese Zeile hier?", „was bedeutet leer?") Sinn ergibt.

## Das Modell

Für jedes (Kind, Datum)-Paar gibt es höchstens einen **Anwesenheits-Datensatz**. Der Datensatz trägt einen Status — derzeit `present` oder `absent`. Das Fehlen eines Datensatzes bedeutet **es wurde keine Beobachtung gemacht**, nicht „das Kind war nicht da".

Dieses Drei-Zustands-Modell (anwesend / abwesend / kein Datensatz) ist wichtig. Eine leere Zelle im Wochenraster ist *nicht* eine ungekennzeichnete Abwesenheit — sie ist „wir wissen es nicht". Berichte, die „Abwesenheits-Tage" zählen, zählen nur Zeilen mit `absent`-Status; Zeilen, die einfach nicht existieren, gehen in keine der Zählungen ein.

## Aktiver-Vertrag-Geltungsbereich

Das Wochen-Anwesenheits-Raster listet nur Kinder, deren Betreuungsvertrag in der angezeigten Woche aktiv ist. Ein Kind, dessen Vertrag nächsten Monat beginnt, erscheint nicht — Anwesenheit für es vor Aufnahme zu erfassen ist ein Datenfehler, den das Modell verhindert.

Wenn Sie das Von-/Bis-Datum eines Kindes ändern, spiegelt das Anwesenheits-Raster die Änderung sofort beim nächsten Refresh.

## Auto-Speichern

Das Raster speichert automatisch bei jedem Zell-Wechsel. Es gibt keine Speichern-Schaltfläche. Implementierung:

1. Klicken Sie eine Zelle an, um durch die Zustände zu schalten.
2. Das Frontend ruft je nach Zustands-Übergang den Create/Update/Delete-Endpunkt auf.
3. Bei 2xx-Antwort zeigt die Zelle den neuen Zustand.
4. Bei Fehler wird die Zelle zurückgesetzt und ein Inline-Fehler erscheint.

Das Fehlen eines explizit Speichern ist Absicht — Erzieher:innen sollten am Ende des Tages keinen Knopf drücken müssen.

## Pro-Kind-Anwesenheits-Historie

Das Wochenraster ist die breite Sicht. Die schmale Sicht ist auf der Detailseite des Kindes, die jeden Anwesenheits-Datensatz dieses Kindes in Datums-Reihenfolge auflistet. Nützlich für Eltern-Berichte („wie viele Tage war Max dieses Halbjahr abwesend?") und um Muster zu erkennen.

## Berichte

Tages-Übersichts-Endpunkte aggregieren die pro-Kind-Datensätze zu Anwesenheits-Zählungen pro Tag. Die Tages-Übersicht des Dashboards nutzt diese. Die Daten sind exakt für jeden Tag, an dem alle Kinder erfasst wurden; sie zählen anwesende Tage zu niedrig für jeden Vergangenheits-Tag, an dem manche Zellen leer gelassen wurden.

## Was bewusst nicht modelliert wird

- **Halbtagsanwesenheit.** Ein Kind ist entweder den ganzen Betreuungstag anwesend oder abwesend.
- **Grund-Codes.** „Abwesend wegen Krankheit" vs. „abwesend wegen Urlaub" wird nicht gespeichert.
- **Bring-/Abholzeiten.** Anwesenheit ist binär pro Tag, nicht zeit-bereich-basiert.

Diese Auslassungen entsprechen der Berliner Kita-Konvention. Wenn ein Bundesland oder eine Organisation reicheres Modellieren braucht, ist es eine Datenmodell-Änderung, keine UI-Änderung.

## Was für Audit erfasst wird

Jedes Anwesenheits-Create/Update/Delete schreibt einen Audit-Log-Eintrag: wer hat was, wann, von welcher IP erfasst. Die Einträge erscheinen im org-bezogenen Audit-Log filterbar nach `attendance_*`-Aktionen.
