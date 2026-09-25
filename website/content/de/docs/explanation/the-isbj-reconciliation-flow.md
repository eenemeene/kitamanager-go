---
title: Der ISBJ-Abgleich
weight: 6
---

Wenn Sie eine ISBJ-Excel über die Förder-Bescheid-Seite hochladen, durchläuft KitaManager eine mehrstufige Pipeline. Diese Seite beschreibt jede Stufe, damit Sie eine Excel debuggen können, die nicht importiert, oder einen Vergleich, der unerwartete Ergebnisse produziert.

## Stufe 1 — Parsen

Die Excel-Datei wird mit `internal/isbj/parse.go` gelesen. Der Parser:

1. Findet das Tabellenblatt mit den pro-Kind-Detail-Zeilen (Tabellenblatt-Namen folgen einer stabilen Senats-Konvention).
2. Liest jede Zeile und normalisiert Spaltennamen gegen eine interne Map.
3. Extrahiert: Familienname/Vorname des Kindes, Gutscheinnummer, abgerechnete Beträge pro Zuschlag sowie Typ und Geltungsmonat der Zeile (Korrekturen — siehe unten).

Parse-Fehler werden inline angezeigt. Häufige Ursachen:

- Das Excel-Layout hat sich in einem Senats-Update geändert — die Spalten-Map des Parsers braucht ein Update.
- Die Datei ist gar kein ISBJ-Bescheid (z. B. eine fremde XLSX wurde hochgeladen).
- Das Tabellenblatt ist leer.

## Stufe 2 — Persistieren

Erfolgreich geparste Zeilen werden an zwei Orten gespeichert:

- Eine `government_funding_bill_period` für den gesamten Upload (deckt den Datumsbereich ab, referenziert die Datei).
- Ein `government_funding_bill_entry` pro Zeile.

Erneutes Hochladen desselben Zeitraums ersetzt den vorherigen — die Einträge der vorherigen Abrechnung werden gelöscht, bevor die neuen eingefügt werden. Audit-Log dokumentiert beide Ereignisse.

## Stufe 3 — Matching

Für jeden Eintrag sucht KitaManager nach einem Kind in der Organisation mit dieser Gutscheinnummer. Das Matching erfolgt nur über `voucher_number` — Namen werden nicht zum Matching genutzt, weil senats-seitige Umbenennungen und Schreibweisen-Unterschiede üblich sind.

- **Match gefunden** → der Eintrag wird mit dem Kind gepaart. KitaManager schaut auf den aktiven Vertrag des Kindes für den Bescheid-Monat.
- **Kein Match** → der Eintrag wird als „Zusätzlich in Abrechnung“ markiert.

Kinder, die einen aktiven Vertrag im Bescheid-Monat haben, deren Gutscheinnummer aber überhaupt nicht in der Abrechnung erscheint, werden als „Fehlt in Abrechnung“ markiert.

## Stufe 4 — Vergleichen

Für jedes gematchte (Eintrag, Kind-Vertrag)-Paar berechnet KitaManager, was *es* für den Vertrag in dem Monat abrechnen würde, mit der Förder-Konfiguration (siehe [Wie Vertragseigenschaften die Förderung bestimmen](../how-contract-properties-determine-funding/)). Es vergleicht dann mit dem abgerechneten Betrag des Eintrags.

- **Übereinstimmung** (innerhalb der Rundung) → die Zeile ist grün.
- **Abweichend** → die Zeile ist rot, mit angezeigter Differenz.

Der Vergleich ist pro Eigenschaft: nicht nur Gesamtbetrag, sondern pro-Zuschlag-Betrag. Ein Kind, bei dem die Gesamtsumme passt, aber die Aufschlüsselung abweicht (z. B. eine Seite hat NdH, die andere Integration A), wird als „abweichend“ mit Detail auf Eigenschafts-Ebene markiert.

## K/A-Marker (Korrekturen)

Echte ISBJ-Bescheide tragen in der Spalte „Monat/ Typ“ zwei Angaben pro Zeile: den **Typ** — „A“ für *Abrechnung* (die reguläre Zeile) oder „K“ für *Korrektur* — und den **Monat**, für den die Zeile gilt.

Diese beiden Angaben meinen nicht dasselbe wie der Monat des Bescheids. Ein Bescheid für den April sieht regelmäßig so aus:

| Typ | Monat | Betrag |
|---|---|---|
| K | 01.25 | 1,02 € |
| K | 02.25 | 1,02 € |
| K | 03.25 | 1,02 € |
| A | 04.25 | 946,50 € |

Der Senat sagt damit: hier ist der April — und der Satz, den wir Ihnen im Januar, Februar und März gezahlt haben, war jeweils um 1,02 € zu niedrig. Rückwirkende Korrekturen entstehen, wenn das Kostenblatt nachträglich neu veröffentlicht wird, ein Integrationsstatus mit einem Datum in der Vergangenheit bewilligt wird oder eine NdH-Angabe nachträglich berichtigt wird.

KitaManager wertet beide Angaben aus und ordnet jede Zeile dem Monat zu, für den sie gilt. Deshalb beantworten zwei Ansichten zwei verschiedene Fragen, und beide sind richtig:

- **Finanzen / Saldo** rechnet nach **Eingangsmonat** — was in diesem Monat tatsächlich geflossen ist, Korrekturen eingeschlossen. Das ist die Frage nach der Liquidität.
- **Abrechnungsvergleich** rechnet nach **Geltungsmonat** — war dieser Monat korrekt gefördert. Die drei Korrekturen oben zählen dort zu Januar, Februar und März, nicht zum April.

{{< callout type="info" >}}
Für Bescheide, die vor diesem Stand importiert wurden, ist der Geltungsmonat nicht gespeichert. Diese Zeilen zählen weiterhin zum Eingangsmonat — die Zahlen ändern sich rückwirkend nicht. Wer die Zuordnung auch für ältere Bescheide möchte, importiert die betreffenden Dateien erneut.
{{< /callout >}}

Betrifft eine Korrektur einen Monat, für den gar kein Bescheid importiert wurde, lässt sich für diesen Monat keine Differenz bilden — es gibt keinen berechneten Wert zum Vergleich. Der Betrag geht deshalb nicht verloren, sondern wird in der Kita-Jahres-Zeile als eigener Hinweis unter der Spalte „Korrektur“ ausgewiesen.

## Abweichungsanalyse: woher die Differenz kommt

Unter jeder Kita-Jahres-Zeile steht die **Abweichungsanalyse** mit einer
Aufschlüsselung nach Kategorie: Satzdifferenzen, Eigenschaftsabweichungen, nur
in der Abrechnung, nur im System. Die vier Kategorien sind überschneidungsfrei
und vollständig — sie ergeben zusammen genau die Differenz aus regulärer
Abrechnung und Berechnung.

Korrekturen gehören bewusst **nicht** dazu. Eine Korrektur ist kein Befund, den
Sie beheben können, sondern eine bereits erfolgte Nachzahlung oder Rückforderung
des Senats. Sie verschiebt aber das Ergebnis des Kita-Jahres, und deshalb steht
sie als eigene Zeile darunter:

```
Summe der Kategorien        + 430,09 €
Korrekturen für diese Monate  − 30,00 €
──────────────────────────────────────
Differenz im Kita-Jahr      + 400,09 €
```

Die letzte Zeile entspricht der Differenz in der Kita-Jahres-Zeile darüber. Die
Korrekturen sind dabei nach **Geltungsmonat** gezählt: eine im August gezahlte
Korrektur für den Juli zählt zum Kita-Jahr des Juli, auch wenn der Bescheid
bereits zum nächsten gehört.

Eine Ausnahme sind Korrekturen für Monate, zu denen keine Abrechnung vorliegt;
sie stehen gesondert als *davon ohne Abrechnung*. Sie bleiben aus der Differenz
im Kita-Jahr heraus, aus demselben Grund, aus dem die Kita-Jahres-Zeile sie
nicht in ihre Korrektur-Spalte aufnimmt: diese Monate tragen nichts zur
berechneten Seite bei, und ihre Korrekturen dagegen zu rechnen ergäbe eine
Unterdeckung in der Größe eines Monats, die niemand schuldet.

Für die operative Triage-Matrix (welches Symptom auf welche Korrektur abbildet) siehe [Abweichung in einer Abrechnung untersuchen](../../how-to/use/investigate-a-bill-discrepancy/).
