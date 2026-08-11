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
   - **Betreuungsart** — die passende auswählen: Halbtag, Teilzeit, Ganztag, Ganztag erweitert.
   - **Zuschläge** — alle zutreffenden ankreuzen: NdH, QM/MSS, Integration A, Integration B. Nicht mehr zutreffende entfernen.
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

Liegt der Stichtag in der Vergangenheit (der Bescheid kommt im August, die Änderung gilt ab 1. Februar), lässt sich das über die Oberfläche derzeit **nicht** eintragen.

Der Grund: Bei einem Vertrag, der vor heute begann, legt KitaManager beim Speichern automatisch einen Folgevertrag **ab heute** an. Ein **Bis**-Datum in der Vergangenheit ergäbe damit einen Vertrag, dessen Ende vor seinem Anfang liegt — das lehnt KitaManager ab („from date must be before or equal to to date“). Auch der umgekehrte Weg — zuerst den neuen Vertrag rückwirkend anlegen — wird abgelehnt, weil er sich mit dem laufenden Vertrag überschneidet.

Was Sie stattdessen tun können:

1. Die Änderung **ab heute** eintragen, wie oben beschrieben. Ab jetzt rechnet KitaManager korrekt.
2. Damit rechnen, dass die Monate zwischen Stichtag und heute weiter mit den alten Werten rechnen und beim Abgleich als Abweichung erscheinen. Beim Prüfen hilft [Abweichung in einer Abrechnung untersuchen](../investigate-a-bill-discrepancy/).

Das ist eine bekannte Einschränkung, kein Bedienfehler. Wenn Sie regelmäßig rückwirkende Bescheide bekommen, ist das der Punkt, den es im Produkt zu lösen gilt.

## Falsches Datum korrigieren (kein neuer Vertrag nötig)

Wenn nur das Start- oder Enddatum falsch ist (Sie haben 1. März eingetragen, der Vertrag begann tatsächlich am 15. März), geht eines von beiden:

- **Bearbeiten-Dialog** — Vertrag öffnen, **Von** / **Bis** ändern, speichern.
- **Zeitleisten-Ansicht** — Tab **Zeitleiste** auf der Vertragsseite öffnen und die Vertragsgrenze ziehen.

## Hinweise

- Eine falsche Betreuungsart oder ein fehlender Zuschlag verursacht stillschweigend eine Abweichung beim nächsten ISBJ-Bescheid. Die Differenz kann hunderte Euro pro Kind und Monat ausmachen — die Änderung sofort eintragen, sobald sie bekannt wird.
- Den alten Vertrag nicht löschen. Beenden bewahrt den historischen Eintrag; Löschen entfernt ihn aus Personal-, Belegungs- und Förderberichten.
- Zur Berechnungskette (Alter × Betreuungsart × Zuschläge → Euro), siehe [Wie Vertragseigenschaften die Förderung bestimmen](../../../explanation/how-contract-properties-determine-funding/).
- Das Protokoll erfasst jede Anlage / Bearbeitung / Löschung eines Vertrags mit Alt → Neu-Werten. Admins können es einsehen: [Protokoll prüfen](../../administer/review-audit-log/).
