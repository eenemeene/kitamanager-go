---
title: Update government funding rates
weight: 3
---

The Berlin Senate publishes new Kostenblatt amounts (typically once a year, on August 1). You want to load the new rates so KitaManager's calculations match the new ISBJ Bescheide. This is a **superadmin-only** operation.

## Steps

### Option 1: Import the official YAML

1. Get the updated YAML — typically one ships in the project repo as `configs/government-fundings/berlin.yaml`.
2. Sign in as superadmin.
3. Open **Settings** → **Government Funding Rates**.
4. Click **Import**.
5. Pick the YAML file. The preview shows what will be created.
6. Confirm.

### Option 2: Edit a property in the UI

For a one-off correction:

1. Open the funding configuration → the relevant period → the relevant entry.
2. Edit the property (`payment` is in cents — €2,494.91 → `249491`).
3. Click **Save**.

## Notes

- Funding rates are **global**, not per-organisation. A change applies to every Kita on the system.
- The format is documented at [Government funding YAML format](../../../reference/data-model/funding-yaml-format/).
- `payment` values are in **cents** (integers) to avoid floating-point precision errors. €2,494.91 → `249491`. €−23 → `-2300`.
- After updating: the next bill comparison uses the new rates from the bill's month onwards. Past bills are unchanged in storage, but their re-comparison would now produce different numbers.
- The audit log records that a funding configuration was edited, with who and when — not the old and new values of each property.
