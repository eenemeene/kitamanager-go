---
title: Keep your KitaManager data current
weight: 3
---

KitaManager only stays useful if the data you've entered keeps reflecting reality. Children move up groups, employees change hours, vouchers are renewed, the Tarifrunde lands a new pay table, rent goes up. This tutorial walks you through a routine maintenance pass — the kind of thing a Kita-Leitung sits down to do once or twice a month, plus once a year for the larger pieces.

You don't need to read this end-to-end. Skim the section headings, find the one you have a real change for, and jump to the linked how-to. By the time you've worked through this page once, you'll know where every routine update lives.

You'll need:

- An admin or manager role in your organization. A few items at the end need superadmin.
- Some changes from real life waiting to be entered — or just follow along on the seeded **Kita Sonnenschein** data.
- About 30 minutes if you read everything.

## The two mental models that make this easy

**1. The dashboard is your task list.** Don't go hunt for things to update — open the dashboard and let it tell you what's drifted. Pending step promotions, children outgrowing their section, children without vouchers, bill mismatches — every routine update has a dashboard widget. If the dashboard is empty of warnings, you're current.

{{< screenshot src="/images/screenshots/dashboard.png" alt="KitaManager dashboard" caption="Top to bottom: KPI cards, warnings that need action, then routine widgets. If a widget is empty, that area is current." >}}

**2. Extend history, don't overwrite it.** Real-world changes have an effective date. The right pattern is almost always: end the current record on the day before, create a new one from the effective date. Editing the existing record overwrites history — bills you've already reconciled silently stop matching, the audit log loses the *before* state. The exception is correcting an input error (you typed `30h` but meant `35h` from day one): then edit in place, because the timeline already reflects what *should* have been there.

With those two in mind, here's the routine.

## 1. Keep employee contracts current

The most common employee changes are TVöD-SuE step promotions (every two years per employee, dashboard widget tells you when), hours changes, and grade changes after qualification. Less common but important: section transfers, contract ends.

The dashboard's **Pending Step Promotions** widget is the canonical signal for the most frequent one — it tells you who's due, what the projected monthly cost delta is, and when.

Recipes:

- [Promote an employee step (Stufenaufstieg)](../../how-to/use/promote-employee-step/) — the widget-driven flow for the regular TVöD step advancement.
- [Update an employee contract on a change](../../how-to/use/update-employee-contract/) — hours, grade, fixed-term end, the general pattern.
- [Move an employee between sections](../../how-to/use/move-employee-between-sections/) — drag-and-drop or advance planning.

## 2. Keep children's care contracts current

Children's care arrangements drift more than employees': care type changes (Halbtag → Ganztag is the classic), supplements come and go (NdH onset when family language changes; Integrationsstatus A or B granted by the Bezirks-Jugendamt), vouchers are renewed yearly, sections change as children grow.

Every one of these silently affects the next ISBJ Bescheid if you don't enter it. The dashboard helps in two ways: **Children Without Vouchers** surfaces missing Gutscheine; **Children Over Section Age Limit** surfaces kids who've outgrown their group.

{{< screenshot src="/images/screenshots/children.png" alt="Children list" caption="Each child shows the calculated funding amount. If a number looks wrong, the contract properties are usually the cause." >}}

Recipes:

- [Update a child's care contract](../../how-to/use/update-child-contract/) — care type and supplement changes, the general pattern.
- [Assign a Kita-Gutschein number](../../how-to/use/assign-a-voucher/) — voucher renewals and corrections.
- [Move children between sections](../../how-to/use/move-children-between-sections/) — drag-and-drop on the Sections page.
- [Record a child's departure](../../how-to/use/record-a-childs-departure/) — when a child leaves entirely.

## 3. Keep section assignments current

Sections (Bereiche) are the groups inside your Kita. Two things keep drifting: who's in each section (children grow, staff rotate), and the sections themselves (a group renames, age limits get tightened, a new section opens).

The **Sections** page handles the day-to-day reassignment via drag-and-drop for both children and pedagogical employees. The section configuration (name, age limits, default) is an admin task done from each section's detail page.

{{< screenshot src="/images/screenshots/sections.png" alt="Sections kanban" caption="Each section column shows children and pedagogical employees. Drag a card to reassign — the running contract is normally closed and a new one in the target section created from today." >}}

Recipes:

- [Move children between sections](../../how-to/use/move-children-between-sections/) — drag-and-drop.
- [Move an employee between sections](../../how-to/use/move-employee-between-sections/) — drag-and-drop for pedagogical staff; manual flow for non-pedagogical.
- [Manage sections (rename, age limits, default)](../../how-to/administer/manage-sections/) — admin-side configuration.

## 4. Keep personal data current

Marriage and divorce change names. You'll occasionally find a birthdate typo when checking against the Kita-Gutschein. Both children and employees use the same dialog — first name, last name, gender, birthdate — opened from the Pencil icon on the list page.

A birthdate correction on a child is **not cosmetic** — it can shift the child into a different age bracket for funding, silently changing the calculated amount for every bill month from the contract start. The recipe has the warning details.

Recipe:

- [Update a child's or employee's personal data](../../how-to/use/update-personal-data/) — the dialog plus the don't-shoot-yourself-in-the-foot warnings on birthdate and name changes.

## 5. Keep the pay plan (Entgelttabelle) current

Every Tarifrunde (negotiated wage round) lands a new TVöD-SuE table — typically one new period per year. Loading the new period takes a single YAML import. From the `from` date onwards, salary cost on the dashboard, financial overview, and forecast all use the new rates. Existing contracts don't need touching.

This is an **admin** task, done once per organization per year (or whenever the table changes). The YAML usually arrives from your provider, a colleague, or a previous KitaManager release.

Recipe:

- [Update the pay plan when TVöD-SuE rates change](../../how-to/administer/update-pay-plan/) — YAML import (fast path) and the manual UI path for one-off corrections.

## 6. Keep budget items current

Budget items are the income and expenses that aren't computed from contracts: parent contributions, rent, donations, one-off costs. They're behind the cumulative balance chart and the forecast — wrong numbers here silently swing both.

The maintenance pattern mirrors contracts: when an amount changes mid-year (rent goes up), end the old entry and add a new one from the effective date. When a category ends, just close the entry. Don't delete — that erases history.

Recipe:

- [Manage budget items](../../how-to/use/manage-budget-items/) — create, add entries, and the mid-year-value-change / category-ends / one-off / input-error patterns.

## 7. (Superadmin) Keep government funding rates current

The Berlin Senate publishes new Kostenblatt amounts once a year, on 1 August. Loading them takes a YAML import similar to the pay plan, except that **funding rates are global, not per-organization** — a change applies to every Kita on the system. This is a **superadmin-only** operation, and the YAML usually ships in the KitaManager repo under `configs/government-fundings/berlin.yaml`.

Recipe:

- [Update government funding rates](../../how-to/operate/update-government-funding-rates/) — YAML import as superadmin.

## A routine you can run on the dashboard

Open the dashboard. For each warning card that isn't empty, click into it and follow the linked recipe. When all warning cards are empty, you're current. That's the whole job.

For the annual cycle (typically June–August): import the new pay plan when the Tarifrunde finalises, then on 1 August the superadmin imports the new funding rates. Both are minutes-of-work, but skipping them means every salary number and every Bescheid comparison drifts until you do.
