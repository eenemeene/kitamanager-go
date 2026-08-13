---
title: Move children between sections
weight: 3
---

You want to move a child from one section to another (e.g. they've outgrown the Nest group).

## Quick way: drag and drop

{{< screenshot src="/images/screenshots/sections.png" alt="Sections view with one column per section" caption="Each section is a column. Drag the child's card into the target column." >}}

1. Click **Sections** in the sidebar.
2. Each section is a column. Grab the child's card and drag it to the target column.
3. Drop. KitaManager ends the running contract with a To date of **yesterday** and creates a new contract in the target section starting **today**, so the history is preserved.

If the child's age doesn't fit the target section, KitaManager still moves them but shows a warning.

### Two exceptions

- **The contract starts today or later.** It is then edited in place, with no new contract — there is no past to preserve yet.
- **The contract has already ended** (To date in the past). Dragging is not the right tool: there is no running contract to carry forward. Use **New contract** to create one in the target section. If instead a past section was *recorded wrongly*, edit that contract in the contract history — that changes it in place and records the old and new value in the audit log.

## Manual way: for a specific change date

When the move takes effect on a planned date rather than today:

1. Open the child from the **Children** list and click the **history** icon to open their contract history.
2. Find the **active** contract, click the **pencil**, and set **To** to the day before the change. **Save**.
3. Click **New contract**, set **From** to the change date, and pick the new **Section**. **Save**.

If the change date is in the past, only this path works — see the note on backdated changes in [Update a child's care contract](../update-child-contract/).

## Notes

- Drag and drop is the right tool for "this child moves up to the next group now". The manual path is for advance planning ("Max moves to Große on August 1").
- The dashboard's **Children Over Section Age Limit** widget surfaces children who've passed their section's max age — moving them is the fix.
- For the section-age model, see [What the staffing key means](../../../explanation/what-the-staffing-key-means/).
