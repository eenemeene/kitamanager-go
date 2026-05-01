---
title: Abweichung in einer Abrechnung untersuchen
weight: 7
---

Sie haben eine ISBJ-Abrechnung hochgeladen und der Vergleich zeigt Abweichungen. Sie wollen wissen, welches Kind Sie ansehen sollen und was zu korrigieren ist.

## Die drei Abweichungs-Kategorien

Wenn Bescheid und KitaManager nicht übereinstimmen, fällt jedes nicht passende Kind in genau eine von:

- **Fehlt in Abrechnung** — KitaManager erwartet Förderung für dieses Kind, der Bescheid enthält es nicht. Häufigste Ursache: die Gutscheinnummer in Ihrem Datensatz ist falsch, oder der Gutschein wurde vom Bezirks-Jugendamt noch nicht verarbeitet.
- **Zusätzlich in Abrechnung** — der Bescheid zahlt für ein Kind, das in KitaManager nicht existiert (oder dessen Vertrag beendet ist). Häufigste Ursache: ein Kind ist gegangen und der Bezirk hat seinen Datensatz nicht aktualisiert, oder das Kind wurde nie in KitaManager angelegt.
- **Abweichende Beträge** — beide Seiten sind sich einig, dass das Kind gefördert werden soll, aber die Beträge weichen ab. Häufigste Ursache: die Vertragseigenschaften (Betreuungsart, Zuschläge) passen nicht zu dem, was der Bezirk hinterlegt hat.

## Was tun pro Kategorie

### Fehlt in Abrechnung

1. Öffnen Sie die Detailseite des Kindes in KitaManager.
2. Bestätigen Sie, dass die Gutscheinnummer mit dem Papier-Gutschein übereinstimmt. Tippfehler hier verursachen >90 % der „Fehlt"-Fälle.
3. Wenn der Gutschein neu ist, geben Sie dem Bezirk einen Abrechnungszyklus, um aufzuholen.
4. Wenn die Gutscheinnummer korrekt und nicht neu ist, kontaktieren Sie das Bezirks-Jugendamt: möglicherweise wurde die Aufnahme nicht verarbeitet.

### Zusätzlich in Abrechnung

1. Notieren Sie die Gutscheinnummer aus der Abrechnungs-Zeile und suchen Sie sie in der Kinderliste.
2. Falls das Kind existiert mit einem in der Vergangenheit beendeten Vertrag, hat der Bezirk den Abgang nicht erfasst. Benachrichtigen Sie ihn.
3. Falls das Kind in KitaManager überhaupt nicht existiert, entscheiden Sie: sollte es existieren (dann Datensatz anlegen), oder bezieht sich der Bescheid auf ein Kind, das Sie nie hatten (dann beim Bezirk reklamieren)?

### Abweichende Beträge

1. Öffnen Sie die Detailseite des Kindes → Verträge.
2. Vergleichen Sie jedes Vertragsfeld mit dem Papier-Gutschein: Betreuungsart, NdH, QM/MSS, Integrationsstatus.
3. Häufigste Drift: NdH ist auf einer Seite gesetzt, auf der anderen nicht. NdH wird vom Bezirk unabhängig von der Gutschein-Erneuerung aktualisiert.
4. Korrigieren Sie den Vertrag in KitaManager (oder kontaktieren Sie den Bezirk, je nachdem welche Seite falsch ist).

## Weitere Fehlerbilder

| Symptom | Wahrscheinliche Ursache | Wo nachsehen |
|---|---|---|
| „Datei konnte nicht geparst werden" | Excel-Layout-Drift oder falscher Dateityp | Spalten-Map des Parsers in `internal/isbj/parse.go`; prüfen, ob die Datei wirklich der Bescheid ist |
| „Keine Bescheide für diesen Monat gefunden" | Upload erzeugte einen anderen Zeitraum als erwartet | Bescheid-Zeitraum-Daten auf der Abrechnungs-Seite |
| Viele „Fehlt in Abrechnung"-Einträge | Gutscheinnummern matchen nicht | Kind-Detailseiten — siehe Schritte oben |
| Viele „Zusätzlich in Abrechnung"-Einträge | Kinder sind gegangen, Senat hat den Abgang nicht verarbeitet | Bezirks-Jugendamt |
| Anhaltende Gesamt-Drift über viele Kinder | Fördersätze veraltet | [Berliner Fördersätze aktualisieren](../../operate/update-government-funding-rates/) |
| Gesamt passt, aber Eigenschafts-Aufschlüsselung weicht ab | Zuschlag nicht synchron | Vertrags-Zuschläge des Kindes vs. Papier-Gutschein |

## Hinweise

- Nach der Korrektur die Abrechnung erneut hochladen. Der Vergleich aktualisiert sich, und die Zeile sollte matchen.
- Für die Berechnung siehe [Wie Vertragseigenschaften die Förderung bestimmen](../../../explanation/how-contract-properties-determine-funding/) und [Wie die Förderung in Berlin funktioniert](../../../explanation/how-funding-works-in-berlin/).
- Für die Parsing-Pipeline siehe [Der ISBJ-Abgleich](../../../explanation/the-isbj-reconciliation-flow/).
