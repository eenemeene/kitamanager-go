---
title: Explanation
weight: 5
---

Explanation is **understanding-oriented**. Pages here are background and rationale: how Berlin funds Kitas, what the staffing key actually computes, why the Kita year runs August to July, what KitaManager does when you upload a Bescheid. Read these once, slowly, when you have time — they shouldn't be needed mid-task.

If you need a *step-by-step* recipe, you want a [how-to](../how-to/). If you need exact numbers or field names, [reference](../reference/) is the right place.

## How the Kita business works (everyone)

{{< cards >}}
  {{< card link="how-funding-works-in-berlin/" title="How funding works in Berlin" subtitle="Bezirks-Jugendamt vs. Senatsverwaltung vs. ISBJ; what NdH, QM/MSS, and Integrationsstatus actually mean; how a Kita-Gutschein turns into a monthly bill." icon="document-text" >}}
  {{< card link="how-contract-properties-determine-funding/" title="How contract properties determine funding" subtitle="The lookup chain from a child's birthdate + care_type + supplements to a monthly euro amount." icon="calculator" >}}
  {{< card link="what-the-staffing-key-means/" title="What the staffing key means" subtitle="FTE requirements, required-vs-available hours, and what the dashboard's coverage percent really says." icon="chart-bar" >}}
  {{< card link="why-the-kita-year-runs-aug-to-jul/" title="Why the Kita year runs Aug–Jul" subtitle="Calendar conventions, balance reset semantics, and why charts shade by Kita year, not calendar year." icon="calendar" >}}
  {{< card link="how-attendance-is-modeled/" title="How attendance is modeled" subtitle="Per-day records, the difference between absent and not-yet-recorded, and why the weekly grid auto-saves." icon="clipboard-check" >}}
  {{< card link="the-isbj-reconciliation-flow/" title="The ISBJ reconciliation flow" subtitle="What happens when you upload an Excel: parse, match by voucher number, compare per property, the three mismatch categories." icon="refresh" >}}
{{< /cards >}}

## How KitaManager itself works (developers and operators)

{{< cards >}}
  {{< card link="why-money-is-stored-as-cents/" title="Why money is stored as cents" subtitle="Floating-point traps and the integer-cents convention used end-to-end. Useful when authoring funding YAMLs or working on financial code." icon="currency-euro" >}}
  {{< card link="architecture/" title="Architecture" subtitle="System overview, RBAC, data flow, the report-pdf sidecar, soft-delete model." icon="cube" >}}
{{< /cards >}}
