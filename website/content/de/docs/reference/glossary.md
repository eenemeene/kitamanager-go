---
title: Glossar
weight: 5
---

Das deutsche Kita- und ISBJ-Vokabular, das Ihnen in KitaManager begegnet. Begriffe, die in anderen Bundesländern anders gebraucht werden, sind markiert.

## Behörden und Verfahren

- **Bezirks-Jugendamt** — Berlin hat 12, eines pro Bezirk. Stellt Kita-Gutscheine aus, verarbeitet Gutschein-Anträge, ist Anlaufstelle für Eltern. Wohin Eltern sich wenden.
- **Senatsverwaltung für Bildung, Jugend und Familie** — Berliner Senatsverwaltung. Legt die Fördersätze fest, die das Land pro Kind und Monat zahlt. Betreibt das ISBJ-Verfahren im Auftrag der Bezirke.
- **ISBJ** — *Integriertes Software-System Berliner Jugendhilfe*. Das Verfahren (und die Software dahinter) für den monatlichen Abrechnungs-Austausch zwischen Kita und Senat. Die Excel-Dateien, die Sie in KitaManager hochladen, kommen daher.
- **Bescheid** — die monatliche Abrechnungsmitteilung aus ISBJ in Excel-Form.
- **Kostenblatt** — die vom Senat veröffentlichte Tabelle der pro-Kind-Sätze. Aktualisierung typischerweise einmal jährlich am 1. August.

## Kinder-seitige Begriffe

- **Kita-Gutschein** — vom Bezirks-Jugendamt ausgestellter Gutschein, der ein Kind für geförderte Kita-Betreuung berechtigt. Die Gutscheinnummer verbindet ein Kind mit einem konkreten Förderbetrag.
- **Gutscheinnummer** — Identifier des Gutscheins; Verbindungsschlüssel zwischen KitaManager und ISBJ-Bescheiden. Ohne sie kann nichts abgerechnet werden.
- **Betreuungsart** — eine von `halbtag` (≤5h), `teilzeit` (≤7h), `ganztag` (≤9h), `ganztag erweitert` (>9h).
- **NdH** — *nichtdeutsche Herkunftssprache*. Familienkommunikationssprache ist nicht Deutsch. Statistischer Indikator, mit dem der Senat zusätzliche Personalstunden zuteilt; die Kita erhält einen kleinen pro-Kind-Zuschlag.
- **QM/MSS** — *Quartiersmanagement / Monitoring Soziale Stadtentwicklung*. Die Kita liegt in einem Berliner Quartiersmanagements- oder Sozialraum-Monitoring-Gebiet. Bezieht sich auf den Standort der Kita, nicht das einzelne Kind.
- **Integrationsstatus A / B** — formelle Klassifikation für Eingliederungshilfe (SGB IX körperliche/geistige/Sinnesbehinderung oder SGB VIII §35a seelische Behinderung). A = erhöhter Förderbedarf, B = erheblich erhöht. KitaManager beschriftet diese als `Integration A` / `Integration B`.
- **Eingliederungshilfe** — der rechtliche Rahmen (SGB IX / SGB VIII §35a), unter dem Integrationsstatus gewährt wird.
- **Elternbeitrag** — Kita-Betreuung in Berlin ist weitgehend kostenlos; der einzige eltern-getragene Posten in KitaManagers Förder-Modell ist der −€23/Monat-Essensbeitrag, der auf alle Betreuungsverträge angewendet wird.

## Mitarbeiter-seitige Begriffe

- **Personalkategorie** — das Vertragsfeld `staff_category` nimmt einen von drei Werten an: `qualified` (Fachkraft — voll qualifiziertes pädagogisches Personal), `supplementary` (Ergänzungskraft — ergänzendes pädagogisches Personal), `non_pedagogical` (Hauswirtschaft, Verwaltung etc.). Steuert, ob der Vertrag pädagogisch für die Personalkennzahl zählt.
- **Entgeltgruppe** — Gehaltsgruppe in TVöD-SuE (z. B. `S 8a`). Mit `Stufe` kombiniert, um das Gehalt nachzuschlagen.
- **Stufe** — Erfahrungsstufe innerhalb einer Entgeltgruppe (1–6 für TVöD-SuE).
- **Stufenaufstieg** — Beförderung in die nächste Stufe. Zeitabhängig laut Tarifvertrag; das Dashboard-Widget zeigt berechtigte Mitarbeitende.
- **TVöD-SuE** — *Tarifvertrag für den öffentlichen Dienst, Sozial- und Erziehungsdienst*. Der Tarifvertrag, unter dem die meisten Berliner Kitas zahlen.
- **VZÄ** — *Vollzeitäquivalent*. Die Einheit für Personalstunden-Bedarf im Förder-YAML.

## System-Begriffe

- **Bereich** — Gruppe innerhalb einer Kita (z. B. Nest, Nestflüchter, Große). Im Englischen heißen sie „sections".
- **Kitajahr** — August → Juli. Anders als das Kalenderjahr. Siehe [Warum das Kita-Jahr von August bis Juli läuft](../../explanation/why-the-kita-year-runs-aug-to-jul/).
- **Personalkennzahl / Personalabdeckung** — verfügbare Personalstunden im Verhältnis zu den Anforderungen aus der Förder-Konfiguration. Siehe [Was die Personalkennzahl bedeutet](../../explanation/what-the-staffing-key-means/).

## Andere Bundesländer (kurz)

KitaManager liefert heute nur das Berliner Förder-Modell. Andere Länder nutzen andere Verfahren — Brandenburg hat KitaServer, Bayern hat Kibig etc. — und andere Zuschlags-Namen. Ein neues Bundesland anzubinden bedeutet, eine `configs/government-fundings/<bundesland>.yaml` zu schreiben; die Form „Nachschlagen über Alter und Eigenschaften" verallgemeinert.
