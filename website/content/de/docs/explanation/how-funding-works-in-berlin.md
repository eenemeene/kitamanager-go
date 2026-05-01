---
title: Wie die Förderung in Berlin funktioniert
weight: 1
---

Die Förderlogik in KitaManager bildet ab, wie Berlin Kitas finanziert. Wenn Sie nur mit der Oberfläche arbeiten, lautet die Kurzfassung: Verträge korrekt eintragen, monatliche ISBJ-Abrechnung hochladen, Abweichungen klären. Die ausführliche Fassung unten erklärt die beteiligten Parteien, die Begriffe, die Sie auf Ihrem Bescheid sehen, und wie aus einem Vertrag ein Eurobetrag pro Monat wird — nützlich, wenn etwas nicht abrechenbar ist und Sie wissen müssen, wen Sie kontaktieren sollten.

## Drei Parteien, eine Abrechnung

Manchmal heißt es: „Das Jugendamt bezahlt das.“ Das ist eine bequeme Kurzform, vermischt aber drei verschiedene Stellen, die in KitaManager unterschiedlich behandelt werden.

| Partei | Rolle |
|---|---|
| **Bezirks-Jugendamt** (12 in Berlin, eines pro Bezirk) | Stellt den **Kita-Gutschein** für Eltern aus, bearbeitet Gutschein-Anträge, beantwortet Eltern-Anfragen. Die „Gutscheinnummer“, die Sie in einen Betreuungsvertrag eintragen, kommt von hier. |
| **Senatsverwaltung für Bildung, Jugend und Familie** | Legt die **Fördersätze** fest (nach Altersgruppe, Betreuungsart und Zuschlägen), die das Land pro Kind und Monat zahlt. Betreibt das **ISBJ-Verfahren** im Auftrag der Bezirke. |
| **ISBJ** — Integriertes Software-System Berliner Jugendhilfe | Das Verfahren (und die zugehörige Software) für den monatlichen Abrechnungs-Austausch zwischen Kita und Senat. Die Excel-Dateien, die Sie in KitaManager hochladen, kommen von hier. |

Also:

- Eltern beantragen einen **Kita-Gutschein** beim **Bezirks-Jugendamt**. Sie übergeben Ihnen die Gutscheinnummer beim Anmelden.
- Die **Senatsverwaltung** legt fest, wie viel Geld die Kita pro Kind erhält, abhängig von den Vertragsdetails.
- Jeden Monat erzeugt das **ISBJ**-Verfahren einen Bescheid (Excel-Datei) mit dem, was der Senat tatsächlich zahlt. KitaManager vergleicht das mit der eigenen Berechnung.

Wenn Eltern ihren Gutschein anfechten, schicken Sie sie zum Bezirks-Jugendamt. Wenn ein Auszahlungsbetrag falsch wirkt, kommen die Sätze von der Senatsverwaltung. Wenn ein Kind unerwartet auf der Abrechnung erscheint oder fehlt, lief die Datenbewegung über ISBJ.

## Die Zuschläge auf einem Vertrag

Drei Zuschläge erhöhen den pro-Kind-Förderbetrag in Berlin. KitaManager bietet alle drei im Vertragsformular an:

- **NdH** — *nichtdeutsche Herkunftssprache*: Familienkommunikationssprache ist nicht Deutsch.
- **QM/MSS** — *Quartiersmanagement / Monitoring Soziale Stadtentwicklung*: die Kita selbst liegt in einem QM/MSS-klassifizierten Gebiet. Bezogen auf den Standort der Kita, nicht das einzelne Kind.
- **Integrationsstatus A / B**: Kind ist für Eingliederungshilfe klassifiziert (SGB IX körperlich/geistig/Sinnes oder SGB VIII §35a seelisch). A = erhöhter Bedarf, B = erheblich erhöht. Klassifikation vom Bezirks-Jugendamt nach Antrag der Eltern.

Jeder Zuschlag bringt zusätzliche Personalstunden und einen höheren pro-Kind-Satz. Für Felddetails (key/value-Paare im YAML, Bedarfsmengen) siehe [Referenz: Vertrags-Zuschläge](../../reference/data-model/contract-supplements/) und das [Glossar](../../reference/glossary/).

## Die Förder-Berechnung Ende-zu-Ende

Wenn KitaManager den monatlichen Förderbetrag eines Kindes berechnet, läuft diese Berechnungskette gegen die Berliner Förder-Konfiguration in `configs/government-fundings/berlin.yaml`:

1. **Aktiven Förderzeitraum finden** für den Abrechnungsmonat — die Tabelle ändert sich, wenn der Senat ein neues Kostenblatt veröffentlicht (typischerweise einmal jährlich am 1. August).
2. **Alter des Kindes in Monaten berechnen** zum Abrechnungsmonat aus dem Geburtsdatum.
3. **Grundsatz nachschlagen** über Altersbereich × Betreuungsart. Betreuungsarten sind `ganztag erweitert` (>9h), `ganztag` (≤9h), `teilzeit` (≤7h), `halbtag` (≤5h).
4. **Zuschlagsbeträge addieren** für jeden aktiven NdH-, QM/MSS-, Integrationsstatus-Zuschlag.
5. **Eltern-Essensbeitrag abziehen** (`parent: meals`, derzeit −23 €/Monat, gilt für alle Betreuungsverträge).

Die resultierende Zahl ist das, was KitaManager als „berechnete Förderung“ für dieses Kind in diesem Monat anzeigt. Der ISBJ-Bescheid macht im Prinzip dieselbe Berechnung auf Senatsseite; die Differenzen sind genau das, was KitaManager beim Abrechnungs-Upload gegenüberstellt.

## Warum Abweichungen entstehen

Wenn die Berechnung von KitaManager nicht zum ISBJ-Bescheid passt, ist die Ursache fast immer eine von:

- **Gutscheinnummer fehlt oder ist falsch** in KitaManager → Kind erscheint als „Fehlt in Abrechnung“. Der Bescheid organisiert alles über die Gutscheinnummer; ein Tippfehler oder leeres Feld bricht das Matching.
- **Zuschläge nicht synchron** zwischen Ihrem Datensatz und dem, was der Senat hat → Kind matcht, aber die Beträge differieren. NdH und QM/MSS werden vom Bezirk häufig auch unterjährig aktualisiert; KitaManager muss nachgepflegt werden.
- **Betreuungsart geändert** (z. B. Eltern haben von Teilzeit auf Ganztag erweitert), aber der Vertrag in KitaManager zeigt noch den alten Wert → Betrag differiert.
- **Kind angemeldet oder ausgetreten** in Ihren Daten, aber der entsprechende Gutschein-Vorgang ist beim Bezirk noch nicht verarbeitet → „Fehlt in Abrechnung“ oder „Zusätzlich in Abrechnung“.
- **Fördersätze veraltet** in KitaManager → systematische Drift über viele Kinder. Aktualisieren über [Berliner Fördersätze aktualisieren](../../how-to/operate/update-government-funding-rates/).

Die ersten beiden sind mit Abstand die häufigsten.
