---
title: Update an employee contract on a change
weight: 18
---

An employee's working conditions change: more (or fewer) weekly hours, a different grade, a different section, or the end of a fixed-term contract. You want to record the change so salary cost and staffing coverage stay correct.

For step promotions specifically, use [Promote an employee step (Stufenaufstieg)](../promote-employee-step/) — that recipe walks the dashboard widget.

## The rule: extend history, don't edit

A real-world change has an effective date — the day the new hours / grade / section start. **End the current contract on the day before, and create a new contract from the effective date onwards.** The old contract documents the prior period; KitaManager uses both for historical reporting, staffing calculations on past months, and the salary chart.

Editing the existing contract instead overwrites history: a payroll figure from three months ago would silently change, and an audit-trail diff would only show "after", not the real change.

The one exception is **correcting an input error** (you typed `30h` but meant `35h` from day one, the start date was a typo). Edit the contract directly in that case — the timeline already reflects the reality you want.

## Steps — record a change going forward

Use this path for hours change, grade change, section transfer, or any contract field where the change has an effective date.

{{< screenshot src="/images/screenshots/employee-contracts.png" alt="Employee contract history" caption="The employee's contract history with row actions on the right." >}}

1. Open the employee from the **Employees** list and click the **history** icon to open their contract history.
2. Find the **active** contract (status badge: *active*). Click the **pencil** to edit.
3. Set **To** to the day before the change takes effect (e.g. change starts on 1 March → To = 28 February). Click **Save**.
4. Back on the contracts page, click **New contract**.
5. Set **From** to the effective date. Copy every field from the old contract, then change only the one that's actually different (hours, grade, etc.).
6. Click **Save**.

{{< screenshot src="/images/screenshots/employee-contract-create.png" alt="New employee contract dialog" caption="Same dialog is used for create and edit — the title differs." >}}

The dashboard's staffing-hours and salary-cost figures update immediately. From the effective date onwards the new values feed staffing coverage, the financial overview, and the forecast.

## Special case: moving an employee to a different section

For a section change, drag-and-drop on the **Sections** page is the quickest path — for a contract that started before today, KitaManager closes the old contract and creates a new one in the target section automatically. The contract edit dialog itself doesn't expose the section field, so you can't change it from this page. See [Move an employee between sections](../move-employee-between-sections/) for both the drag-and-drop and the advance-planning recipes.

## Special case: contract end (fixed-term, departure, parental leave)

The employee's contract is ending and they're not getting a new one immediately. Just set the **To** date on the active contract and save — no new contract needed. From the day after **To**, the employee stops contributing to staffing requirements and salary cost.

The employee record stays on file with no active contract, which is correct: historical reports still reference the prior contract.

## Fix a wrong date (no new contract needed)

If only the start or end **date** is wrong (you wrote March 1 when the contract really started March 15), use one of:

- **Edit dialog** — open the contract, change **From** / **To**, save.
- **Timeline view** — switch to the **Timeline** tab on the contracts page and drag the contract boundary to the right date. Useful when you want to see all contracts at once.

## Notes

- Step promotions have their own widget and recipe — don't follow the manual steps here for that. See [Promote an employee step](../promote-employee-step/).
- A wrong grade/step/hours combination silently miscomputes salary cost. Double-check the new contract on the salary chart (the chart updates as soon as you save).
- The audit log records every contract create / edit / delete with old → new values. Admins can review it via [Review the audit log](../../administer/review-audit-log/).
- For the salary calculation chain (grade × step × hours × pay-plan-rate), see the admin-side recipe [Update the pay plan](../../administer/update-pay-plan/).
