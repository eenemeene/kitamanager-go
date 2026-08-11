---
title: Move an employee between sections
weight: 20
---

You want to reassign an employee to a different section (e.g. a teacher moving from the Nest group to the Große group).

## Quick way: drag and drop

{{< screenshot src="/images/screenshots/sections.png" alt="Sections kanban board" caption="Employees appear on the section card alongside children — drag the card to the target section." >}}

1. Click **Sections** in the sidebar.
2. Each section is a column showing children and pedagogical employees. Find the employee's card and drag it to the target section column.
3. Drop. For a contract that started before today, KitaManager closes the old contract (To = yesterday) and creates a new one in the target section starting today. For a contract that started today or later, KitaManager updates the section in place.

If the contract has **already ended** (To date in the past), KitaManager rejects the change. Use **New contract** to create one in the target section instead of dragging.

## Manual way: edit the contract

When you need to set a specific change date (not "today") — for advance planning, or for a non-pedagogical employee who doesn't appear on the sections board:

1. Open the employee from the **Employees** list and click the **history** icon.
2. Find the **active** contract. Click the **pencil** to edit.
3. Set **To** to the day before the move (move on 1 March → To = 28 February). Click **Save**.
4. Click **New contract**.
5. Set **From** to the move date. Pick the new **Section**. Copy every other field (pay plan, staff category, grade, step, weekly hours) from the old contract.
6. Click **Save**.

## Notes

- The Sections board only shows **pedagogical** employees (Fachkraft, Hilfskraft, Leitung). Non-pedagogical staff (Hauswirtschaft, etc.) don't appear there — use the manual way for them.
- The contract edit dialog itself does not expose a section field. That's deliberate: changing the section on a contract that started in the past would silently rewrite historical staffing reports. That's why the drag-and-drop and manual paths normally produce a new contract from the effective date, so the history stays correct.
- Drag-and-drop is the right tool for "this employee moves to the next group today". The manual path is for "Anna moves to Große on August 1".
- The staffing-coverage calculation switches at the move date. The dashboard's staffing widget updates immediately.
- For the broader contract-change workflow (hours, grade, step), see [Update an employee contract](../update-employee-contract/).
- For the staffing-coverage calculation, see [What the staffing key means](../../../explanation/what-the-staffing-key-means/).
