---
title: Betreuungsvertrag eines Kindes bei einer Änderung aktualisieren
weight: 19
---

Die Situation eines Kindes ändert sich: Die Betreuungszeit wird länger (Halbtag → Ganztag), ein Zuschlag beginnt oder endet (NdH greift nach Wechsel der Familiensprache, Integrationsstatus A oder B wurde vom Bezirks-Jugendamt anerkannt) oder die Eltern bringen einen neuen Kita-Gutschein. Sie wollen die Änderung erfassen, damit der nächste ISBJ-Bescheid weiterhin zu dem passt, was KitaManager berechnet.

Diese Anleitung deckt Änderungen an Betreuungsart und Zuschlägen ab. Für die anderen Aktualisierungs-Abläufe gibt es eigene Anleitungen:

- Neue Gutscheinnummer → [Kita-Gutschein-Nummer zuweisen](../assign-a-voucher/)
- Bereichswechsel → [Kinder zwischen Bereichen verschieben](../move-children-between-sections/)
- Vertragsende (Kind verlässt die Kita) → [Abgang eines Kindes erfassen](../record-a-childs-departure/)

## Die Regel: Historie erweitern, nicht überschreiben

Betreuungsart und Zuschläge steuern den berechneten monatlichen Förderbetrag (siehe [Wie Vertragseigenschaften die Förderung bestimmen](../../../explanation/how-contract-properties-determine-funding/)). Wenn Sie sie am bestehenden Vertrag bearbeiten, **werden alle vergangenen Monate stillschweigend gegen die neuen Werte neu berechnet** — bereits abgestimmte Bescheide passen plötzlich nicht mehr zu dem, was KitaManager heute berechnet, und das Protokoll zeigt nur den „neuen“ Zustand.

Stattdessen: Eine echte Änderung hat ein Stichtag-Datum. Beenden Sie den aktuellen Vertrag am Tag davor. Legen Sie einen neuen Vertrag ab dem Stichtag mit den neuen Eigenschaften an.

Ausnahme: **Eingabefehler korrigieren** — Sie hatten `halbtag` eingetragen, gemeint war `ganztag` von Anfang an. Den Vertrag bearbeiten; die Zeitleiste spiegelt dann den korrekten Zustand.

## Schritte — Änderung der Betreuungsart oder Zuschläge zum Stichtag

{{< screenshot src="/images/screenshots/child-contracts.png" alt="Vertragshistorie eines Kindes" caption="Die Vertragshistorie zeigt Betreuungsart und Zuschläge als Badges." >}}

1. Öffnen Sie das Kind über die **Kinder**-Liste und klicken Sie auf das **Verlaufs**-Symbol, um die Vertragshistorie zu öffnen.
2. Suchen Sie den **aktiven** Vertrag (Status-Badge: *aktiv*). Klicken Sie auf den **Stift** zum Bearbeiten.
3. Setzen Sie **Bis** auf den Tag vor dem Stichtag (z. B. Integrationsstatus A ab 1. Februar → Bis = 31. Januar). **Speichern**. Das **Bis**-Datum muss heute oder später liegen — für einen Stichtag in der Vergangenheit siehe [Rückwirkende Änderungen](#rückwirkende-änderungen).
4. Zurück in der Vertragshistorie klicken Sie auf **Neuer Vertrag**.
5. Setzen Sie **Von** auf das Stichtag-Datum. Wählen Sie denselben **Bereich** (es sei denn, der Bereich ändert sich ebenfalls — dann siehe [Kinder zwischen Bereichen verschieben](../move-children-between-sections/)).
6. Setzen Sie die **Eigenschaften** des Vertrags:
   - **Betreuungsart** — die passende auswählen: Halbtag (bis 5h), Teilzeit (bis 7h), Ganztag (bis 9h), Ganztag Erweitert (>9h).
   - **Zuschläge** — alle zutreffenden ankreuzen: NdH, QM/MSS, Integration A, Integration B. Nicht mehr zutreffende entfernen.
   - Das Badge **Essen** setzen Sie nicht selbst. Der Eltern-Essensbeitrag ist in der Förder-Konfiguration für alle Verträge hinterlegt und wird automatisch angewendet — er erscheint an jedem Vertrag und lässt sich hier nicht abwählen.
7. **Speichern**.

{{< screenshot src="/images/screenshots/child-contract-create.png" alt="Dialog Neuer Betreuungsvertrag" caption="Betreuungsart wählen und jeden zutreffenden Zuschlag ankreuzen." >}}

Der berechnete monatliche Förderbetrag des Kindes aktualisiert sich sofort — prüfen Sie den neuen Betrag in der Kinderliste, bevor Sie weitermachen.

## Typische Änderungsszenarien

| Anlass | Was ändern |
|---|---|
| Betreuungsdauer ändert sich (Halbtag → Ganztag etc.) | **Betreuungsart** in den Eigenschaften |
| Familie wechselt die Kommunikationssprache auf nicht-Deutsch | **NdH**-Zuschlag hinzufügen |
| Kita wird in ein QM/MSS-Gebiet eingestuft | **QM/MSS**-Zuschlag hinzufügen |
| Bezirks-Jugendamt erkennt Eingliederungshilfe an | **Integration A** oder **Integration B** hinzufügen |
| Zuschlag läuft aus oder wird aberkannt | Den Zuschlag entfernen |
| Gutscheinverlängerung oder neue Gutscheinnummer | Siehe [Kita-Gutschein zuweisen](../assign-a-voucher/) — gleicher Vertrag, kein Beenden+Neuanlegen nötig |

## Rückwirkende Änderungen

Liegt der Stichtag in der Vergangenheit (der Bescheid kommt im August, die Änderung gilt ab 1. Februar), geht das in zwei Schritten:

1. **Änderung normal erfassen**, wie oben beschrieben. KitaManager beendet den alten Vertrag gestern und legt den neuen ab heute an — noch nicht das gewünschte Datum.
2. **Zeitleiste öffnen** (Tab **Zeitleiste** auf der Vertragsseite) und die Grenze zwischen den beiden Verträgen auf den Stichtag ziehen. Der alte Vertrag endet dann am Tag davor, der neue beginnt am Stichtag.

Danach rechnet KitaManager auch die vergangenen Monate ab dem Stichtag mit den neuen Werten.

Warum zwei Schritte: Im Bearbeiten-Dialog eines Vertrags, der vor heute begann, legt KitaManager beim Speichern immer einen Folgevertrag **ab heute** an — ein **Bis**-Datum in der Vergangenheit ergäbe dort einen Vertrag, dessen Ende vor seinem Anfang liegt, und wird abgelehnt. Die Zeitleiste arbeitet direkt an den bestehenden Verträgen und darf deshalb in die Vergangenheit greifen.

Die Grenze lässt sich nur innerhalb der Nachbarverträge verschieben — nicht vor den Beginn des älteren Vertrags.

## Falsches Datum korrigieren (kein neuer Vertrag nötig)

Wenn nur das Start- oder Enddatum falsch ist (Sie haben 1. März eingetragen, der Vertrag begann tatsächlich am 15. März), geht eines von beiden:

- **Bearbeiten-Dialog** — Vertrag öffnen, **Von** / **Bis** ändern, speichern.
- **Zeitleisten-Ansicht** — Tab **Zeitleiste** auf der Vertragsseite öffnen und die Vertragsgrenze ziehen.

## Hinweise

- Eine falsche Betreuungsart oder ein fehlender Zuschlag verursacht stillschweigend eine Abweichung beim nächsten ISBJ-Bescheid. Die Differenz kann hunderte Euro pro Kind und Monat ausmachen — die Änderung sofort eintragen, sobald sie bekannt wird.
- Den alten Vertrag nicht löschen. Beenden bewahrt den historischen Eintrag; Löschen entfernt ihn aus Personal-, Belegungs- und Förderberichten.
- Zur Berechnungskette (Alter × Betreuungsart × Zuschläge → Euro), siehe [Wie Vertragseigenschaften die Förderung bestimmen](../../../explanation/how-contract-properties-determine-funding/).
- Das Protokoll erfasst jede Vertragsänderung **mit dem alten und dem neuen Wert** der geänderten Felder — Datum, Bereich, Betreuungsart und Zuschläge. Wirkt ein Förderbetrag falsch, zeigt das Protokoll, was sich wann geändert hat. Eine Änderung ohne Wirkung wird ohne Werte erfasst. Admins können es einsehen: [Protokoll prüfen](../../administer/review-audit-log/).
