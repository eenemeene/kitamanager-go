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

Vouchers are stored as a list per child — issuing a new Gutschein doesn't replace the old one; both stay on file. Add the new voucher number through the same dashboard input or via the API (`POST /organizations/{orgId}/children/{childId}/vouchers`). The most recent voucher matches against current bills; older vouchers stay on the child's record for historical reconciliation.

The assignment is idempotent: re-submitting an already-known voucher number is a no-op, no error.

## Notes

- Voucher numbers must be unique across the organisation.
- If you already have ISBJ bills uploaded for past months, the comparison for those months won't change — the bill data is frozen at upload time. Re-upload the relevant Bescheide if you need the matches retroactively.
- For the broader investigation flow when a bill won't match, see [Investigate a bill discrepancy](../investigate-a-bill-discrepancy/).
- For the per-child billing view (which voucher hit which Bescheid), open **Billing history** on the child's detail page.
