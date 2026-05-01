---
title: Der ISBJ-Abgleich
weight: 6
---

Wenn Sie eine ISBJ-Excel über die Förder-Bescheid-Seite hochladen, durchläuft KitaManager eine mehrstufige Pipeline. Diese Seite beschreibt jede Stufe, damit Sie eine Excel debuggen können, die nicht importiert, oder einen Vergleich, der unerwartete Ergebnisse produziert.

## Stufe 1 — Parsen

Die Excel-Datei wird mit `internal/isbj/parse.go` gelesen. Der Parser:

1. Findet das Tabellenblatt mit den pro-Kind-Detail-Zeilen (Tabellenblatt-Namen folgen einer stabilen Senats-Konvention).
2. Liest jede Zeile und normalisiert Spaltennamen gegen eine interne Map.
3. Extrahiert: Familienname/Vorname des Kindes, Gutscheinnummer, abgerechnete Beträge pro Zuschlag, K/A-Marker (Korrekturen — siehe unten).

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
- **Kein Match** → der Eintrag wird als „Zusätzlich in Abrechnung" markiert.

Kinder, die einen aktiven Vertrag im Bescheid-Monat haben, deren Gutscheinnummer aber überhaupt nicht in der Abrechnung erscheint, werden als „Fehlt in Abrechnung" markiert.

## Stufe 4 — Vergleichen

Für jedes gematchte (Eintrag, Kind-Vertrag)-Paar berechnet KitaManager, was *es* für den Vertrag in dem Monat abrechnen würde, mit der Förder-Konfiguration (siehe [Wie Vertragseigenschaften die Förderung bestimmen](../how-contract-properties-determine-funding/)). Es vergleicht dann mit dem abgerechneten Betrag des Eintrags.

- **Übereinstimmung** (innerhalb der Rundung) → die Zeile ist grün.
- **Abweichend** → die Zeile ist rot, mit angezeigter Differenz.

Der Vergleich ist pro Eigenschaft: nicht nur Gesamtbetrag, sondern pro-Zuschlag-Betrag. Ein Kind, bei dem die Gesamtsumme passt, aber die Aufschlüsselung abweicht (z. B. eine Seite hat NdH, die andere Integration A), wird als „abweichend" mit Detail auf Eigenschafts-Ebene markiert.

## K/A-Marker (Korrekturen)

Echte ISBJ-Bescheide tragen „K"- (Korrektur) und „A"- (Aufhebung) Marker auf Zeilen, die rückwirkend einen vorherigen Monat korrigieren oder stornieren. KitaManagers Parser **ignoriert diese Marker derzeit** und behandelt jede Zeile als eigenständig für den Bescheid-Monat. Das verursacht eine bekannte Bescheid-Vergleichs-Drift, wenn der Senat Beträge aus einem früheren Monat korrigiert: der korrigierte Betrag wird gegen den Monat verbucht, in dem der Bescheid *ausgestellt* wurde, nicht den Monat, für den er *gilt*, sodass zwei Monate gegenläufige Differenzen statt einer passenden Zeile zeigen. Bekannte Einschränkung; die Umgehung beim Triagieren ist, gegenläufige Differenzen über aufeinanderfolgende Monate zu ignorieren.

Für die operative Triage-Matrix (welches Symptom auf welche Korrektur abbildet) siehe [Abweichung in einer Abrechnung untersuchen](../../how-to/use/investigate-a-bill-discrepancy/).
