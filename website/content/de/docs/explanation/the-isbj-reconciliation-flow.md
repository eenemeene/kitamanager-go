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
Aufschlüsselung nach Kategorie: Satzdifferenzen, Eigenschaftsabweichungen,
Monat nicht abgerechnet, nur in der Abrechnung, nur im System. Die fünf
Kategorien sind überschneidungsfrei und vollständig — sie ergeben zusammen
genau die Differenz aus regulärer Abrechnung und Berechnung.

**Monat nicht abgerechnet** ist die Kategorie, die man kennen sollte. Das Kind
steht in der Abrechnung, aber nur mit Korrekturen für frühere Monate: für den
eigenen Monat der Abrechnung gibt es keine reguläre Zeile. Der Betrag ist ein
vollständiger Monatssatz, und er ist echt — dieser Monat wurde nicht gefördert.
Was er *nicht* ist: ein Kostenblatt-Problem. Nach Eigenschaften aufgeschlüsselt
las es sich als „care_type: ganztag — im Vertrag, aber nicht abgerechnet“, was
Sie beim Senat über einen völlig korrekten Betreuungsumfang diskutieren lässt.
Die richtige Frage ist, warum das Kind nicht abgerechnet wurde — meist eine noch
laufende Registrierung, die ein oder zwei Monate später rückwirkend
abgerechnet wird.

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

### Manuelle Korrekturbuchungen

Die Abrechnungsübersicht führt eine zweite, eigenständige Größe: die
**Korrekturbuchung manuell**. Das ist eine Buchung, die der Senat auf
Bezirksebene von Hand vornimmt, und ihr liegen keine Kinderzeilen zugrunde —

```
tatsächlicher Zahlbetrag gesamt = Vertragsbuchung + Korrekturbuchung manuell
```

— wobei die kindbezogenen Zeilen der Vertragsübersicht genau die
Vertragsbuchung ergeben. Jede kindbezogene Größe in KitaManager ist dafür also
von Natur aus blind, und die Buchung braucht eine eigene Zeile:

```
Summe der Kategorien          + 430,09 €
Korrekturen für diese Monate   − 30,00 €
Korrekturbuchungen (manuell) + 18.230,85 €
──────────────────────────────────────────
Differenz im Kita-Jahr        + 400,09 €
```

Sie bleibt aus der Differenz im Kita-Jahr heraus, aus demselben Grund wie die
Korrekturen ohne Abrechnung: es gibt keinen berechneten Wert, dem man sie
gegenüberstellen könnte. In der Praxis handelt es sich um
*Nachfinanzierung I-Status* — ein rückwirkend bewilligter Integrationsstatus —
und die Buchungen betreffen Monate ein bis zwei Kita-Jahre zurück. Sie dem
Eingangsjahr zuzurechnen würde dieses deutlich zu gut aussehen lassen.

Welche Monate eine Buchung betrifft, steht ausschließlich im Blatt
**Korrekturübersicht**, als Freitext. KitaManager liest daraus die Monate und die
Art der Buchung und weist sie neben dem Betrag aus — eine Kita-Jahres-Zeile
nennt dann „zzgl. 18.230,85 € manuelle Korrekturbuchung für 10/23 – 09/24“
statt eines unerklärten Betrags, der wie Einnahme dieses Jahres aussieht.

Nicht gespeichert wird der Freitext selbst. Der Korrekturgrund nennt Kind und
Geburtsdatum; gespeichert werden ausschließlich die Monate und ein kurzer
Grund-Code, und weder Datenbank noch API führen den Wortlaut. Ergeben die Zeilen
des Blattes nicht die dort ausgewiesene Korrektursumme — ein Layout, das der
Parser noch nicht kennt —, bleibt der Betrag bestehen und die Monate werden als
unbekannt ausgewiesen, statt geraten zu werden.

## Ein Kind, zwei Gutscheinzeilen in einer Abrechnung

Bei einer Gutschein-Verlängerung stellt der Senat nicht einfach um. Er **storniert**
die unter dem alten Gutschein bereits gezahlten Monate — als negative
Korrekturzeilen — und **bucht sie neu** unter dem neuen Gutschein, der zugleich
die reguläre Zeile für den eigenen Monat der Abrechnung trägt. Ein Kind
erscheint damit zweimal in derselben Abrechnung, unter zwei Gutscheinnummern,
und die beiden Zeilen sind ein einziges Betreuungsverhältnis.

KitaManager vergleicht je Kind, nicht je Gutscheinzeile, sodass der berechnete
Monatssatz genau einmal zählt. Der Vergleich nennt den Gutschein, auf dem das
Geld tatsächlich eingegangen ist, und führt beide auf.

Für die operative Triage-Matrix (welches Symptom auf welche Korrektur abbildet) siehe [Abweichung in einer Abrechnung untersuchen](../../how-to/use/investigate-a-bill-discrepancy/).
