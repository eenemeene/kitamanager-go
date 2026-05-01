---
title: Erklärungen
weight: 5
---

Erklärungen sind **verständnis-orientiert**. Seiten hier sind Hintergrund und Begründung: wie Berlin Kitas finanziert, was die Personalkennzahl wirklich berechnet, warum das Kita-Jahr von August bis Juli läuft, was KitaManager beim Hochladen einer Bescheid-Datei tut. Lesen Sie diese einmal, in Ruhe, wenn Sie Zeit haben — sie sollten nicht mitten in einer Aufgabe nötig sein.

Wenn Sie ein Schritt-für-Schritt-*Rezept* brauchen, suchen Sie in [How-to](../how-to/). Für genaue Zahlen oder Feldnamen ist die [Referenz](../reference/) richtig.

## Wie das Kita-Geschäft funktioniert (für alle)

{{< cards >}}
  {{< card link="how-funding-works-in-berlin/" title="Wie die Förderung in Berlin funktioniert" subtitle="Bezirks-Jugendamt vs. Senatsverwaltung vs. ISBJ; was NdH, QM/MSS und Integrationsstatus wirklich bedeuten; wie ein Kita-Gutschein zu einer monatlichen Abrechnung wird." icon="document-text" >}}
  {{< card link="how-contract-properties-determine-funding/" title="Wie Vertragseigenschaften die Förderung bestimmen" subtitle="Die Berechnungskette von Geburtsdatum + care_type + Zuschlägen zum monatlichen Eurobetrag." icon="calculator" >}}
  {{< card link="what-the-staffing-key-means/" title="Was die Personalkennzahl bedeutet" subtitle="VZÄ-Bedarf, benötigte vs. verfügbare Stunden, und was die Personalabdeckungs-Prozentzahl auf dem Dashboard wirklich aussagt." icon="chart-bar" >}}
  {{< card link="why-the-kita-year-runs-aug-to-jul/" title="Warum das Kita-Jahr von August bis Juli läuft" subtitle="Kalenderkonventionen, Saldo-Reset-Semantik und warum Diagramme nach Kita-Jahr und nicht nach Kalenderjahr schattieren." icon="calendar" >}}
  {{< card link="how-attendance-is-modeled/" title="Wie Anwesenheit modelliert wird" subtitle="Tagesgenaue Datensätze, Unterschied zwischen abwesend und nicht-erfasst, warum das Wochengitter automatisch speichert." icon="clipboard-check" >}}
  {{< card link="the-isbj-reconciliation-flow/" title="Der ISBJ-Abgleich" subtitle="Was beim Hochladen einer Excel passiert: Parsen, Matching nach Gutscheinnummer, Vergleich pro Eigenschaft, die drei Abweichungs-Kategorien." icon="refresh" >}}
{{< /cards >}}

## Wie KitaManager selbst funktioniert (für Entwickler:innen und Betreiber:innen)

{{< cards >}}
  {{< card link="why-money-is-stored-as-cents/" title="Warum Geldbeträge als Cents gespeichert werden" subtitle="Floating-Point-Fallen und die Konvention ganzzahliger Cents im gesamten System. Hilfreich beim Schreiben von Förder-YAMLs oder bei Finanzcode." icon="currency-euro" >}}
  {{< card link="architecture/" title="Architektur" subtitle="Systemüberblick, RBAC, Datenfluss, das report-pdf-Sidecar, Soft-Delete-Modell." icon="cube" >}}
{{< /cards >}}
