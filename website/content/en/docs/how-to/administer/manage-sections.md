---
title: Manage sections (Bereiche)
weight: 10
---

Sections are the groups within your Kita (Nest, Nestflüchter, Große, etc.). You want to create, rename, configure age limits, set the default for new contracts, or remove a section.

## Create a section

1. Click **Sections** in the sidebar.
2. Click **Create**.
3. Set **Name**, optional **Min age (months)** and **Max age (months)**.
4. Click **Save**.

The section appears as a column on the Sections page and as an option when creating contracts.

## Rename or change age limits

Click the section header → edit fields → **Save**. Existing contracts that reference the section keep doing so; the new name shows everywhere immediately.

If you tighten the age range, children currently in the section may now exceed the new max age — the dashboard's **Children Over Section Age Limit** widget will surface them.

## Set the default section

When creating a contract, the **Section** field pre-selects whichever section is marked default. To change which one is default:

1. Open the section detail page.
2. Click **Promote to default**.

Only one section per organisation is the default at a time; promoting a new one demotes the previous default.

## Delete a section

Click **Delete** on the section detail page. The section is removed only if no contracts (active or historical) reference it; otherwise the operation is blocked. Reassign children/employees to a different section first using [Move children between sections](../../use/move-children-between-sections/) or by editing employee contracts.

## Notes

- Sections are organisation-scoped. Multi-org users see different section lists per organisation.
- The min/max age values feed two things: the dashboard age-overflow warning, and the staffing-requirement calculation in [What the staffing key means](../../../explanation/what-the-staffing-key-means/).
