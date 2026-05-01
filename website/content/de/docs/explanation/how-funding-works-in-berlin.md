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

Drei Zuschläge erhöhen den pro-Kind-Förderbetrag in Berlin. KitaManager bietet alle drei im Vertragsformular an. Die deutschen Begriffe, die Sie auf Ihrem Bescheid sehen, bedeuten sehr Spezifisches:

### NdH — *nichtdeutsche Herkunftssprache*

Setzen Sie diesen Zuschlag, wenn die Familienkommunikationssprache **nicht überwiegend Deutsch** ist. Die offizielle Senats-Definition bezieht sich auf die *Herkunftssprache* der Familie — nicht auf die Haushaltszusammensetzung, nicht auf die Staatsangehörigkeit. NdH ist ein statistischer Indikator; der Senat verteilt darüber zusätzliche Personalstunden, und die Kita erhält einen kleinen Zuschlag pro Kind.

### QM/MSS — *Quartiersmanagement / Monitoring Soziale Stadtentwicklung*

Setzen Sie diesen Zuschlag, wenn **die Kita selbst in einem QM/MSS-klassifizierten Gebiet liegt**. Der Zuschlag wird Kitas in Quartiersmanagements-Gebieten oder in Gebieten gezahlt, die im Sozialraum-Monitoring entsprechend ausgewiesen sind. Das hängt *nicht* vom einzelnen Kind ab — es hängt von der Lage der Kita und der sozialen Zusammensetzung der betreuten Kinder ab (in Kombination mit NdH greift der Zuschlag, wenn mehr als 40 % der Kinder NdH-Status haben). Wenn Sie nicht wissen, ob Ihre Kita in einem QM/MSS-Gebiet liegt, kann das Bezirks-Jugendamt Auskunft geben.

### Integrationsstatus A / B

Setzen Sie diesen Zuschlag, wenn das Kind formell für **Eingliederungshilfe** klassifiziert ist — nach SGB IX (körperliche, geistige, Sinnesbehinderung) oder SGB VIII §35a (seelische Behinderung). Die Klassifikation — A für erhöhten Förderbedarf, B für erheblich erhöhten — kommt vom Bezirks-Jugendamt nach gesondertem Antrag der Eltern. Beide Status bringen sowohl zusätzliche Personalstunden als auch einen höheren pro-Kind-Satz mit sich.

(KitaManager beschriftet diese als `Integration A` und `Integration B`; der offizielle Berliner Begriff lautet `Integrationsstatus A/B` bzw. `A-Status / B-Status`. Die Eingliederungshilfe ist die rechtliche Grundlage; der Berliner Kita-spezifische Status sitzt darauf.)

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

## Andere Bundesländer

Das Förder-Modell von KitaManager ist datengetrieben: die Sätze und Eigenschaften liegen in YAML, nicht im Code. Ein neues Bundesland anzubinden bedeutet, eine `configs/government-fundings/<bundesland>.yaml` mit dessen Satzstruktur zu schreiben und zu importieren. Heute liegt nur `berlin.yaml` mit dem Projekt; weitere Bundesländer sind auf der Roadmap.

Die anderen Bundesländer verwenden andere Verfahren (Brandenburg hat KitaServer, Bayern hat Kibig, etc.), daher unterscheiden sich Bescheid-Format und Zuschlags-Namen — aber die Form „Nachschlagen über Alter und Eigenschaften“ ist allgemein genug, sie zu erfassen.
