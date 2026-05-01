---
title: Datenmodell
weight: 3
---

Das KitaManager-Datenmodell liegt in PostgreSQL. Das vollständige ER-Diagramm und eine Tabellen-Referenz werden von [`tbls`](https://github.com/k1LoW/tbls) automatisch generiert und liegen im Repository unter [`docs/schema/`](https://github.com/eenemeene/kitamanager-go/tree/main/docs/schema). Sie werden bei jeder Schemaänderung neu generiert; die maßgebliche Quelle ist die laufende Datenbank, nicht diese Markdown-Dateien.

Für YAML-Formate, die von Import-Endpunkten verwendet werden:

{{< cards >}}
  {{< card link="funding-yaml-format/" title="Förder-YAML-Format" subtitle="Die Form einer Fördersatz-YAML: Konfigurationen, Zeiträume, Eigenschaften, Zahlbeträge." icon="document-text" >}}
  {{< card link="contract-supplements/" title="Vertrags-Zuschläge" subtitle="Die key/value-Paare, die Sie an einen Betreuungsvertrag eines Kindes anhängen können (NdH, QM/MSS, Integration A/B), und die Betreuungsarten." icon="document-text" >}}
{{< /cards >}}

Für die Designentscheidungen (warum Geld in Cents gespeichert wird, warum Nutzer:innen + Organisationen soft-deleted sind), siehe [Erklärungen](../../explanation/).
