---
title: How funding works in Berlin
weight: 1
---

KitaManager's funding logic models how Berlin funds Kitas. If you only deal with the UI, the short version is: enter your contracts correctly, upload the monthly ISBJ bill, fix any mismatches. The longer version below explains the parties involved, the terminology you'll see in your Bescheid, and how a child's contract turns into a euro amount per month — useful when something doesn't reconcile and you need to know who to talk to.

## Three parties, one bill

People sometimes say "the Jugendamt pays for it". That's a useful shorthand but it conflates three distinct entities. Each has a different role, and KitaManager treats them differently.

| Party | Role |
|---|---|
| **Bezirks-Jugendamt** (12 in Berlin, one per district) | Issues the **Kita-Gutschein** to parents, processes voucher applications, answers parent questions. The "Gutscheinnummer" you enter on a child contract comes from here. |
| **Senatsverwaltung für Bildung, Jugend und Familie** (Senate Department for Education, Youth and Family) | Sets the funding **rates** (per age group, care type, and supplements) that the state pays per child per month. Operates the **ISBJ procedure** on behalf of the districts. |
| **ISBJ** — Integriertes Software-System Berliner Jugendhilfe | The procedure (and the software behind it) for the monthly bill exchange between Kita and the Senate. The Excel files you upload to KitaManager come from here. |

So:

- Parents apply for a **Kita-Gutschein** at their **Bezirks-Jugendamt**. They give you the voucher number when they enrol their child.
- The **Senatsverwaltung** sets how much money the Kita gets per child, depending on the contract details.
- Each month the **ISBJ** procedure produces a Bescheid (Excel file) listing what the Senate is actually paying you. KitaManager compares that to its own calculation.

If a parent disputes their voucher, send them to the Bezirks-Jugendamt. If a payment amount looks wrong, the rate table comes from the Senatsverwaltung. If a child appears or disappears unexpectedly on the bill, the data flowed through ISBJ.

## The supplements you'll see on a contract

Three supplements (Zuschläge) raise the per-child funding amount in Berlin. KitaManager exposes all three on the care-contract form. The German names you see in your Bescheid mean very specific things:

### NdH — *nichtdeutsche Herkunftssprache*

Set this when the family's primary communication language is **not German**. The official Senate definition is the *Herkunftssprache* (heritage / origin language) of the family, not the household composition or the child's citizenship. NdH is a statistical indicator and the Senate uses it to allocate extra staffing hours; the Kita receives a small per-child supplement when NdH is set.

### QM/MSS — *Quartiersmanagement / Monitoring Soziale Stadtentwicklung*

Set this when **the Kita itself sits in a QM/MSS-classified neighbourhood**. The supplement is paid to Kitas in areas designated by Berlin's neighbourhood-management programme or its social-monitoring index. This is *not* about the individual child — it's about where the Kita is located and the social composition of the children it serves (combined with NdH it kicks in when more than 40% of children have NdH status). If you don't know whether your Kita is in a QM/MSS area, your district's Jugendamt can tell you.

### Integrationsstatus A / B

Set this when the child has been formally classified for **Eingliederungshilfe** (integration support) under SGB IX (physical, intellectual, sensory disability) or SGB VIII §35a (mental health). The classification — A for increased support need, B for significantly increased — comes from the Bezirks-Jugendamt after a separate application by the parents. Each status comes with both extra staffing-hour funding and a higher per-child rate.

(KitaManager labels these `Integration A` and `Integration B`; the Berlin official term is `Integrationsstatus A/B` or `A-Status / B-Status`. The underlying Eingliederungshilfe is the legal basis; the Berlin Kita-specific classification sits on top of it.)

## The funding lookup, end to end

When KitaManager calculates a child's monthly funding amount, it runs this lookup against the Berlin funding configuration in `configs/government-fundings/berlin.yaml`:

1. **Find the active funding period** for the bill month — the table changes when the Senate publishes a new Kostenblatt (typically once a year, on August 1).
2. **Compute the child's age in months** at the bill month from their birthdate.
3. **Look up the base rate** by age range × care_type. Care types are `ganztag erweitert` (>9h), `ganztag` (≤9h), `teilzeit` (≤7h), `halbtag` (≤5h).
4. **Add supplement amounts** for any active NdH, QM/MSS, Integrationsstatus.
5. **Subtract the parent meal contribution** (`parent: meals`, currently −23€/month, applied to all care contracts).

The resulting number is what KitaManager displays as "calculated funding" for that child. The ISBJ Bescheid is the same calculation done by the Senate; differences are exactly the comparison KitaManager surfaces on the bill upload page.

## Why mismatches happen

When KitaManager's calculation doesn't match the ISBJ Bescheid, the cause is almost always one of:

- **Voucher number missing or wrong** in KitaManager → child appears as "missing from bill". The bill organises everything by Gutscheinnummer, so a typo or empty field breaks the match.
- **Supplements out of sync** between your record and what the Senate has on file → the child matches but the amounts differ. NdH and QM/MSS in particular are often updated mid-year by the district; you have to keep KitaManager in sync.
- **Care type changed** (e.g. parents extended from Teilzeit to Ganztag) but the contract in KitaManager still shows the old value → amount differs.
- **Child enrolled or left** in your records but the corresponding voucher event hasn't been processed by the district yet → "missing from bill" or "extra in bill".
- **Funding rates outdated** in KitaManager → systematic drift across many children. Update via [Update government funding rates](../../how-to/operate/update-government-funding-rates/).

The first two are by far the most common.

## Other German states

KitaManager's funding model is data-driven: the rates and properties live in YAML, not in code. Adding a new state means writing a `configs/government-fundings/<state>.yaml` with that state's rate structure and importing it. Today only `berlin.yaml` ships with the project; other states are on the roadmap.

The other Bundesländer use different procedures (Brandenburg has KitaServer, Bayern has Kibig, etc.), so the bill upload format and the supplement names will differ — but the lookup-by-age-and-properties shape is general enough to cover them.
