---
title: KitaManager-Daten aktuell halten
weight: 3
---

KitaManager bleibt nur nützlich, solange die eingetragenen Daten weiterhin die Realität spiegeln. Kinder wechseln Gruppen, Mitarbeiter:innen ändern Stunden, Gutscheine werden verlängert, die Tarifrunde landet eine neue Entgelttabelle, die Miete steigt. Dieses Tutorial führt durch einen Routine-Wartungsdurchlauf — die Art Aufgabe, die eine Kita-Leitung ein- bis zweimal im Monat angeht, plus einmal im Jahr für die größeren Posten.

Sie müssen das nicht von vorne bis hinten lesen. Überfliegen Sie die Abschnittsüberschriften, finden Sie diejenige, die zu Ihrer realen Änderung passt, und springen Sie zur verlinkten How-to-Anleitung. Wenn Sie diese Seite einmal durchgearbeitet haben, wissen Sie, wo jede Routine-Aktualisierung lebt.

Sie brauchen:

- Eine Admin- oder Manager-Rolle in Ihrer Organisation. Wenige Punkte am Ende erfordern Superadmin.
- Echte Änderungen, die noch eingetragen werden müssen — oder folgen Sie einfach mit den Beispieldaten der **Kita Sonnenschein**.
- Etwa 30 Minuten, wenn Sie alles lesen.

## Zwei Denkmodelle, die alles einfach machen

**1. Das Dashboard ist Ihre Aufgabenliste.** Suchen Sie nicht aktiv nach Dingen, die zu aktualisieren wären — öffnen Sie das Dashboard und lassen Sie es Ihnen sagen, was abgedriftet ist. Ausstehende Stufenaufstiege, Kinder über der Bereichs-Altersgrenze, Kinder ohne Gutschein, Bescheid-Abweichungen — jede Routine-Aktualisierung hat ein Dashboard-Widget. Ist das Dashboard frei von Warnungen, sind Sie aktuell.

{{< screenshot src="/images/screenshots/dashboard.png" alt="KitaManager-Dashboard" caption="Von oben nach unten: KPI-Karten, Warnungen, die Aktion brauchen, dann Routine-Widgets. Leere Widgets = dieser Bereich ist aktuell." >}}

**2. Historie erweitern, nicht überschreiben.** Reale Änderungen haben ein Stichtag-Datum. Das richtige Muster ist fast immer: aktuellen Eintrag am Tag davor beenden, neuen ab dem Stichtag anlegen. Den bestehenden Eintrag zu bearbeiten überschreibt die Historie — bereits abgestimmte Bescheide passen plötzlich nicht mehr, das Protokoll verliert den *Vorher*-Zustand. Ausnahme: Eingabefehler korrigieren (Sie haben `30h` getippt, gemeint waren `35h` von Anfang an): dann an Ort und Stelle bearbeiten, weil die Zeitleiste dann den korrekten Zustand spiegelt.

Mit diesen beiden im Kopf, hier die Routine.

## 1. Arbeitsverträge aktuell halten

Die häufigsten Mitarbeiter-Änderungen sind TVöD-SuE-Stufenaufstiege (alle zwei Jahre pro Mitarbeiter:in, Dashboard-Widget meldet), Stundenänderungen und Entgeltgruppen-Wechsel nach Qualifikation. Seltener, aber wichtig: Bereichswechsel, Vertragsende.

Das Dashboard-Widget **Ausstehende Stufenaufstiege** ist das kanonische Signal für die häufigste Variante — es zeigt, wer fällig ist, welche monatliche Kostenwirkung prognostiziert wird und wann.

Anleitungen:

- [Stufenaufstieg dokumentieren](../../how-to/use/promote-employee-step/) — der widget-gesteuerte Ablauf für den regulären TVöD-Aufstieg.
- [Arbeitsvertrag bei einer Änderung aktualisieren](../../how-to/use/update-employee-contract/) — Stunden, Entgeltgruppe, Ende einer Befristung, das allgemeine Muster.
- [Mitarbeiter:in zwischen Bereichen verschieben](../../how-to/use/move-employee-between-sections/) — Drag-and-Drop oder Planung im Voraus.

## 2. Betreuungsverträge der Kinder aktuell halten

Die Betreuungssituationen der Kinder driften stärker als die der Mitarbeiter:innen: Wechsel der Betreuungsart (Halbtag → Ganztag ist der Klassiker), Zuschläge kommen und gehen (NdH greift, wenn die Familiensprache wechselt; Integrationsstatus A oder B durch das Bezirks-Jugendamt anerkannt), Gutscheine werden jährlich verlängert, Bereiche ändern sich, wenn Kinder größer werden.

Jede dieser Änderungen beeinflusst still den nächsten ISBJ-Bescheid, wenn Sie sie nicht eintragen. Das Dashboard hilft zweifach: **Kinder ohne Gutschein** weist auf fehlende Gutscheine hin; **Kinder über Bereichs-Altersgrenze** auf Kinder, die ihre Gruppe überaltert haben.

{{< screenshot src="/images/screenshots/children.png" alt="Kinderliste" caption="Jedes Kind zeigt seinen berechneten Förderbetrag. Sieht eine Zahl falsch aus, sind meist die Vertragseigenschaften die Ursache." >}}

Anleitungen:

- [Betreuungsvertrag eines Kindes aktualisieren](../../how-to/use/update-child-contract/) — Betreuungsart- und Zuschlags-Änderungen, das allgemeine Muster.
- [Kita-Gutschein-Nummer zuweisen](../../how-to/use/assign-a-voucher/) — Verlängerungen und Korrekturen.
- [Kinder zwischen Bereichen verschieben](../../how-to/use/move-children-between-sections/) — Drag-and-Drop auf der Bereiche-Seite.
- [Abgang eines Kindes erfassen](../../how-to/use/record-a-childs-departure/) — wenn ein Kind ganz geht.

## 3. Bereichszuordnungen aktuell halten

Bereiche sind die Gruppen in Ihrer Kita. Zwei Dinge driften: wer in welchem Bereich ist (Kinder werden größer, Personal rotiert), und die Bereiche selbst (eine Gruppe wird umbenannt, Altersgrenzen werden geändert, ein neuer Bereich kommt dazu).

Die **Bereiche**-Seite übernimmt die tägliche Neuzuordnung per Drag-and-Drop für Kinder und pädagogische Mitarbeiter:innen. Die Bereichs-Konfiguration (Name, Altersgrenzen, Standard) ist eine Admin-Aufgabe an der Detailseite jedes Bereichs.

{{< screenshot src="/images/screenshots/sections.png" alt="Bereiche-Kanban" caption="Jede Bereichsspalte zeigt Kinder und pädagogische Mitarbeiter:innen. Eine Karte ziehen, um neu zuzuordnen — der Vertrag wird geschlossen und ein neuer im Zielbereich automatisch angelegt." >}}

Anleitungen:

- [Kinder zwischen Bereichen verschieben](../../how-to/use/move-children-between-sections/) — Drag-and-Drop.
- [Mitarbeiter:in zwischen Bereichen verschieben](../../how-to/use/move-employee-between-sections/) — Drag-and-Drop für pädagogisches Personal; manueller Ablauf für nicht-pädagogisches.
- [Bereiche verwalten (umbenennen, Altersgrenzen, Standard)](../../how-to/administer/manage-sections/) — Admin-seitige Konfiguration.

## 4. Personendaten aktuell halten

Heirat und Scheidung ändern Namen. Gelegentlich entdecken Sie einen Tippfehler im Geburtsdatum beim Abgleich mit dem Kita-Gutschein. Kinder und Mitarbeiter:innen nutzen denselben Dialog — Vorname, Nachname, Geschlecht, Geburtsdatum — geöffnet über das Stift-Symbol auf der Listenseite.

Eine Geburtsdatums-Korrektur an einem Kind ist **nicht kosmetisch** — sie kann das Kind in einen anderen Altersbereich für die Förderung verschieben und damit den berechneten Betrag für jeden Bescheid-Monat ab Vertragsbeginn still verändern. Die Anleitung hat die Warn-Details.

Anleitung:

- [Personendaten eines Kindes oder einer Mitarbeiter:in aktualisieren](../../how-to/use/update-personal-data/) — der Dialog plus die Warnhinweise zu Geburtsdatum- und Namensänderungen.

## 5. Entgelttabelle aktuell halten

Jede Tarifrunde landet eine neue TVöD-SuE-Tabelle — typischerweise ein neuer Zeitraum pro Jahr. Den neuen Zeitraum zu laden braucht einen einzelnen YAML-Import. Ab dessen `from`-Datum nutzen Gehaltskosten auf Dashboard, Finanzübersicht und Prognose die neuen Sätze. Bestehende Verträge bleiben unberührt.

Das ist eine **Admin**-Aufgabe, einmal pro Organisation pro Jahr (oder wann immer die Tabelle wechselt). Das YAML kommt üblicherweise von Ihrer Bezugsstelle, einer Kolleg:in oder einem KitaManager-Release.

Anleitung:

- [Entgelttabelle aktualisieren bei TVöD-SuE-Änderungen](../../how-to/administer/update-pay-plan/) — YAML-Import (Schnellweg) und der manuelle UI-Weg für punktuelle Korrekturen.

## 6. Budgetposten aktuell halten

Budgetposten sind die Einnahmen und Ausgaben, die nicht aus Verträgen berechnet werden: Elternbeiträge, Miete, Spenden, Einmalkosten. Sie stehen hinter dem kumulierten Saldo-Diagramm und der Prognose — falsche Zahlen hier verschieben beide still.

Das Wartungsmuster spiegelt das von Verträgen: Wenn sich ein Betrag unterjährig ändert (Miete steigt), den alten Eintrag beenden und einen neuen ab dem Stichtag anlegen. Wenn eine Kategorie endet, einfach den Eintrag schließen. Nicht löschen — das löscht die Historie.

Anleitung:

- [Budgetposten verwalten](../../how-to/use/manage-budget-items/) — Anlegen, Einträge hinzufügen und die Muster „Wert ändert sich unterjährig" / „Kategorie endet" / „Einmalzahlung" / „Eingabefehler".

## 7. (Superadmin) Fördersätze aktuell halten

Die Berliner Senatsverwaltung veröffentlicht einmal pro Jahr, am 1. August, neue Kostenblatt-Werte. Sie zu laden ähnelt dem Pay-Plan-Import, allerdings **sind Fördersätze global, nicht pro Organisation** — eine Änderung wirkt auf jede Kita im System. Das ist eine **superadmin-exklusive** Aktion, und das YAML wird üblicherweise im KitaManager-Repo unter `configs/government-fundings/berlin.yaml` mitgeliefert.

Anleitung:

- [Fördersätze aktualisieren](../../how-to/operate/update-government-funding-rates/) — YAML-Import als Superadmin.

## Eine Routine, die Sie am Dashboard fahren können

Dashboard öffnen. Für jede Warn-Karte, die nicht leer ist, hineinklicken und der verlinkten Anleitung folgen. Wenn alle Warn-Karten leer sind, sind Sie aktuell. Das ist die ganze Aufgabe.

Für den Jahreszyklus (typischerweise Juni–August): Die neue Entgelttabelle importieren, wenn die Tarifrunde abgeschlossen ist; am 1. August importiert der Superadmin die neuen Fördersätze. Beides ist Minuten-Arbeit, aber Auslassen bedeutet, dass jede Gehaltszahl und jeder Bescheid-Vergleich driftet, bis Sie es tun.
