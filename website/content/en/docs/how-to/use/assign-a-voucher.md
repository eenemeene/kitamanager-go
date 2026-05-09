---
title: Assign a Kita-Gutschein number
weight: 15
---

A child shows up under "Children Without Vouchers" on the dashboard, or you got a fresh Kita-Gutschein from the Bezirks-Jugendamt. You want to attach the voucher number so KitaManager can reconcile this child against the next ISBJ Bescheid.

## Steps (from the dashboard)

1. Open the dashboard. The **Children Without Vouchers** card lists every child with a contract but no voucher number.
2. For each row, paste the Gutscheinnummer from the paper Kita-Gutschein into the inline input.
3. Press **Enter** (or click the save icon). The child immediately drops out of the warning list.

If the dashboard suggests a name match (the bill referenced "Müller, Maria" but you entered "Maria Mueller"), an **Accept suggestion** button appears next to the suggested rename. Use it to align the names in one click.

## Add another voucher (renewal or correction)

Vouchers are stored as a list per child — issuing a new Gutschein doesn't replace the old one; both stay on file.

### From the children list (recommended)

1. Open **Children** in the sidebar.
2. Find the child and click the **Vouchers** action button (ticket icon) in the row.
3. The Vouchers dialog opens with all currently-assigned numbers. Type the new Kita-Gutschein number in the format `GB-DDDDDDDDDDD-NN` (11 digits, dash, 2-digit suffix) and click **Add**.
4. To remove a wrong number (e.g. typo correction), click the trash icon next to the entry and confirm. The unique slot is freed, so the same number can then be assigned to a different child afterwards.

### From the dashboard

The **Children Without Vouchers** card on the dashboard remains the fastest path for the *first* assignment after creating a child or uploading a bill. Once a voucher exists on a child, use the dialog above for any subsequent additions or corrections.

### Via the API

`POST /organizations/{orgId}/children/{childId}/vouchers` (add) and `DELETE /organizations/{orgId}/children/{childId}/vouchers/{voucherId}` (remove). The most recent voucher matches against current bills; older vouchers stay on the child's record for historical reconciliation.

Re-submitting the same voucher to the same child is a no-op. Submitting a voucher already attached to a different child returns a conflict; remove it from the previous child first using the Vouchers dialog.

## Add a child the bill knows about but KitaManager doesn't

If the Bezirks-Jugendamt is billing for a Kita-Gutschein that isn't assigned to anyone in KitaManager, the dashboard surfaces it under **Children Only In The Bill**. This is the inverse of *Children Without Vouchers* — the bill has the child, you don't.

1. Click **Add to KitaManager** on the row.
2. The dialog opens with name, birthdate, and contract start pre-filled from the bill data.
3. **Verify the birthdate against the Kita-Gutschein paperwork before saving.** The bill format only carries month and year, so the day defaults to the 1st of the first-seen billing month — that's almost certainly wrong as a real birthday and *will* skew the school-enrollment classification ("Muss-Kind") and any age-based statistics if left as-is.
4. Pick a section and gender (these aren't in the bill data).
5. Click **Create child** — the dialog creates the child, an initial contract starting on the bill's first-seen month, and assigns the voucher number, in that order.

If the voucher is already on a different child globally (extremely unusual — the constraint is global), the create-child step still succeeds but the voucher assignment returns a 409. The dialog surfaces the conflict so you know to clean it up via the Vouchers dialog on the other child first.

## Who can do what

- **View voucher numbers**: every role with access to the child (admin, manager, member, staff).
- **Add or remove voucher numbers**: admins and managers only. Members and staff see the list read-only — the Add input and Remove buttons are hidden for them.

## Notes

- Voucher numbers are globally unique (matching the Bezirks-Jugendamt's own numbering).
- If you already have ISBJ bills uploaded for past months, the comparison for those months won't change — the bill data is frozen at upload time. Re-upload the relevant Bescheide if you need the matches retroactively.
- For the broader investigation flow when a bill won't match, see [Investigate a bill discrepancy](../investigate-a-bill-discrepancy/).
- For the per-child billing view (which voucher hit which Bescheid), open **Billing history** on the child's detail page.
