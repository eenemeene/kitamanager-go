---
title: Update a child's or employee's personal data
weight: 21
---

You want to correct or update personal data — first name, last name, gender, or birthdate. Maybe a family handed in a marriage certificate, or a colleague legally changed their name, or you noticed a birthdate typo when comparing against the Kita-Gutschein.

The personal-data edit dialog is the same for children and employees. It only edits these four fields; contracts, vouchers, attendance, and billing history are not affected.

## Update a child's personal data

{{< screenshot src="/images/screenshots/children.png" alt="Children list" caption="Each row has a pencil icon — Edit personal data." >}}

1. Open the **Children** list.
2. Find the child and click the **pencil** icon in the row.
3. Change the fields you need:
   - **First name** / **Last name** — written exactly as on the Kita-Gutschein.
   - **Gender** — used only for statistics, not for funding.
   - **Birthdate** — see the warning below before changing this.
4. Click **Save**.

{{< screenshot src="/images/screenshots/child-edit-personal.png" alt="Edit child personal data" caption="The personal-data dialog — four fields, nothing else." >}}

## Update an employee's personal data

Same dialog, same fields, opened from the **Employees** list via the pencil icon. The dialog title changes to *Edit employee*; everything else is identical.

{{< screenshot src="/images/screenshots/employees.png" alt="Employees list" caption="Pencil per row — same dialog as for children." >}}

## Warning — birthdate corrections silently change funding

The child's birthdate decides which age-bracket entry in the government-funding configuration applies (see [How contract properties determine funding](../../../explanation/how-contract-properties-determine-funding/)). Changing a birthdate from, say, *2022-07-15* to *2022-08-15* may shift the child from one age bracket to the next at a different month, changing the calculated funding for every bill month from the contract start onwards.

Before saving a birthdate change:

- Cross-check against the Kita-Gutschein paper. The bill's matching algorithm only sees month+year, so a small typo (wrong day) usually doesn't break the match — but a wrong month or year will.
- Check whether the child is now near the school-enrollment threshold ("Muss-Kind" boundary) for your state. KitaManager auto-fills the contract end date from birthdate + state on the contract dialog; if you change the birthdate, **previously-issued contracts still have the old end date** — review them and adjust if needed.

For an employee, the birthdate only affects display — it doesn't change pay or staffing.

## Warning — name changes can break ISBJ auto-matching

If the next ISBJ Bescheid references "Müller, Maria" and you've renamed the record to "Mueller, Maria", the auto-match falls back to the voucher number. The dashboard surfaces this as a *name mismatch suggestion* on the **Children Without Vouchers** card or in the bill comparison — accept the suggestion to re-align.

Avoid creative spelling. Mirror the Kita-Gutschein paper exactly, including hyphens, umlauts, and double-barrelled names.

## What this dialog does NOT change

- **Contracts** — care type, supplements, hours, section, dates. Use [Update a child's care contract](../update-child-contract/) or [Update an employee contract](../update-employee-contract/).
- **Vouchers** — see [Assign a Kita-Gutschein](../assign-a-voucher/).
- **Attendance** — historical attendance entries are not rewritten; the new name shows on the attendance grid from now on.

## Notes

- The audit log captures every personal-data change with the old → new value. Admins can review it via [Review the audit log](../../administer/review-audit-log/).
- Deleting and re-creating the record is **not** an alternative — deletion erases attendance, contract, and billing history. Always edit in place.
- Gender is a closed list (male, female, diverse) in line with the standard German civil-status options.
