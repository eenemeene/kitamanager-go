---
title: Glossary
weight: 5
---

The German Kita and ISBJ vocabulary you'll encounter in KitaManager. Terms used differently in other Bundesländer are noted.

## Authorities and procedures

- **Bezirks-Jugendamt** — district youth office. Berlin has 12, one per Bezirk. Issues Kita-Gutscheine, processes voucher applications, handles parent-facing voucher questions. Where parents go.
- **Senatsverwaltung für Bildung, Jugend und Familie** — Berlin Senate department for education, youth, and family. Sets the funding rates the state pays per child per month. Operates the ISBJ procedure on behalf of the districts.
- **ISBJ** — *Integriertes Software-System Berliner Jugendhilfe*. The procedure (and the software behind it) for the monthly bill exchange between Kita and the Senate. The Excel files you upload to KitaManager come from here.
- **Bescheid** — the monthly billing notice from ISBJ, in Excel form.
- **Kostenblatt** — the Senate's published table of per-child rates. Updated typically once a year on August 1.

## Children-side terms

- **Kita-Gutschein** — voucher issued by the Bezirks-Jugendamt that authorises a child for funded Kita care. The Gutscheinnummer ties a child to a specific funding amount.
- **Gutscheinnummer** — the voucher's identifier; the join key between KitaManager and ISBJ Bescheide. Without it, no funding can be billed.
- **Betreuungsart** — care type. One of `halbtag` (≤5h), `teilzeit` (≤7h), `ganztag` (≤9h), `ganztag erweitert` (>9h).
- **NdH** — *nichtdeutsche Herkunftssprache*. The family's primary communication language is not German. Statistical indicator the Senate uses to allocate extra staffing hours; the Kita receives a small per-child supplement.
- **QM/MSS** — *Quartiersmanagement / Monitoring Soziale Stadtentwicklung*. The Kita itself sits in a Berlin neighbourhood-management or social-monitoring area. About the Kita's location, not the individual child.
- **Integrationsstatus A / B** — formal classification for Eingliederungshilfe (SGB IX physical/intellectual/sensory disability or SGB VIII §35a mental health). A = increased need, B = significantly increased. KitaManager labels these `Integration A` and `Integration B`.
- **Eingliederungshilfe** — the legal framework (SGB IX / SGB VIII §35a) under which Integrationsstatus is granted.
- **Elternbeitrag** — parent contribution. Berlin Kita care is largely free; the only parent-paid line in KitaManager's funding model is the −€23/month meal contribution applied to all care contracts.

## Employees-side terms

- **Personalkategorie** — staff category. The contract `staff_category` field takes one of three values: `qualified` (Fachkraft — fully qualified pedagogical staff), `supplementary` (Ergänzungskraft — supplementary pedagogical staff), `non_pedagogical` (Hauswirtschaft, Verwaltung, etc.). Drives whether the contract counts as pedagogical for the staffing key.
- **Entgeltgruppe** — pay grade in TVöD-SuE (e.g. `S 8a`). Combined with `Stufe` to look up the salary.
- **Stufe** — experience step within a pay grade (1–6 for TVöD-SuE).
- **Stufenaufstieg** — promotion to the next step. Time-based per the collective agreement; the dashboard widget surfaces employees who are eligible.
- **TVöD-SuE** — *Tarifvertrag für den öffentlichen Dienst, Sozial- und Erziehungsdienst*. The collective bargaining agreement that most Berlin Kitas pay under.
- **VZÄ** — *Vollzeitäquivalent*. Full-time-equivalent. The unit for staffing-hour requirements in the funding YAML.

## System terms

- **Bereich** — section / group within a Kita (e.g. Nest, Nestflüchter, Große). KitaManager calls these "sections" in English.
- **Kitajahr** — Kita year, August → July. Different from calendar year. See [Why the Kita year runs Aug–Jul](../../explanation/why-the-kita-year-runs-aug-to-jul/).
- **Personalkennzahl / Personalabdeckung** — staffing key / staffing coverage. Available staff hours vs. required hours from the funding configuration. See [What the staffing key means](../../explanation/what-the-staffing-key-means/).

## Other Bundesländer (briefly)

KitaManager today ships only the Berlin funding model. Other states use different procedures — Brandenburg has KitaServer, Bayern has Kibig, etc. — and different supplement names. Adding a new state means writing a `configs/government-fundings/<state>.yaml`; the lookup-by-age-and-properties shape generalises.
