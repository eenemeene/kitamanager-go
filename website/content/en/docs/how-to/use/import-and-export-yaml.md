---
title: Import or export YAML data
weight: 14
---

You want to bulk-load or export children, employees, or pay plans without clicking through the UI.

## Import

1. Navigate to the relevant page — **Children**, **Employees**, **Pay Plans**, or **Funding rates**.
2. Click **Import**.
3. Select the YAML file. KitaManager reads it and shows a preview.
4. Review the preview, then confirm. Records are created.

Supported imports:

| Data | Format |
|---|---|
| Children | YAML |
| Employees | YAML |
| Pay plans | YAML |
| Government funding rates | YAML (superadmin only — see [Update government funding rates](../../operate/update-government-funding-rates/)) |

## Export

1. Navigate to the relevant page.
2. Click **Export** and pick the format.

Supported exports:

| Data | Formats |
|---|---|
| Children | Excel, YAML |
| Employees | Excel, YAML |
| Pay plans | YAML |

Excel exports open in Excel, LibreOffice Calc, and Google Sheets without conversion.

## Notes

- **Imports create new records.** They do not update or merge with existing ones — duplicate Gutscheinnummern would create two children. For updates, edit each record in the UI or via API.
- For the YAML shape of funding-rate files, see [Government funding YAML format](../../../reference/data-model/funding-yaml-format/).
- For automation (CI exports, batch loads), call the API directly — see [API: Children](../../../reference/api/#children) and friends.
