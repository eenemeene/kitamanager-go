---
title: Bulk-import children and employees from YAML
weight: 11
---

You have a list of children or employees to load — typically the first time you migrate from spreadsheets, or for a whole new section starting in August.

## Steps

1. Navigate to **Children** or **Employees** in the sidebar.
2. Click **Import**.
3. Select the YAML file. KitaManager parses it and shows a preview of which records would be created.
4. Review carefully. **Imports create, they don't merge** — duplicate Gutscheinnummern or duplicate (first_name, last_name, birthdate) tuples will create duplicates that you'd then have to clean up by hand.
5. Confirm.

## Easiest path: export, edit, import

The cleanest way to author a YAML file is to **export the existing data first** (Children → Export → YAML, or Employees → Export → YAML), copy the structure, and edit the values for the new records. The exporter emits exactly the shape the importer accepts; no field-name guessing.

## Minimal YAML shape — children

```yaml
children:
  - first_name: Max
    last_name: Mustermann
    gender: male                          # male | female | diverse
    birthdate: '2024-03-15'
    vouchers:
      - '4711-2026-08-AB'                 # list at child level (not on contract)
    contracts:
      - from: '2026-08-01'
        to: null                          # null = open-ended
        section_name: Nest                # by name
        properties:
          care_type: ganztag
          ndh: ndh                        # omit if not applicable
```

## Minimal YAML shape — employees

```yaml
employees:
  - first_name: Anna
    last_name: Muster
    gender: female
    birthdate: '1990-06-21'
    contracts:
      - from: '2026-08-01'
        to: null
        staff_category: qualified         # qualified | supplementary | non_pedagogical
        grade: S8a
        step: 3
        weekly_hours: 39
        payplan_name: TVöD-SuE 2024       # by name
        section_name: Nest
```

## Verification after import

- Open **Children** / **Employees** and spot-check 3 random records: are the contract dates, sections, and properties what you expected?
- Open the dashboard. **Children Without Vouchers** should be empty if you populated all `vouchers` lists.
- Open **Statistics → Staffing Hours** for the import month. Required and available hours should reflect the new headcount.

## Notes

- Pay plans and sections are looked up by name. They must exist in the organisation before the import. If you're starting from a fresh organisation, set up sections and import the pay plan first.
- For the funding-rate YAML format (different — superadmin-only), see [Government funding YAML format](../../../reference/data-model/funding-yaml-format/) and [Update government funding rates](../../operate/update-government-funding-rates/).
- For exporting back out, see [Import or export YAML data](../../use/import-and-export-yaml/).
