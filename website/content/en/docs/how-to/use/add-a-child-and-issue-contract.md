---
title: Add a child and issue a care contract
weight: 2
---

You want to enroll a new child and capture the care contract that determines their funding.

## Steps

### Create the child record

1. Click **Children** in the sidebar.
2. Click **Create**.
3. Fill in **First name**, **Last name**, **Gender**, **Birthdate**.
4. Click **Save**.

### Issue the care contract

1. The new child opens. Scroll to the **Contracts** section.
2. Click **Create Contract**.
3. Set the basic fields:
   - **From** — when the contract starts (typically the child's first day).
   - **To** — when it ends (leave empty for open-ended).
   - **Voucher number (Gutscheinnummer)** — the number from the Kita-Gutschein the parents received from the Bezirks-Jugendamt. **Without this, no funding can be billed.**
   - **Section** — which group the child joins.
4. Set the contract properties:
   - **Care type (Betreuungsart)** — Halbtag, Teilzeit, Ganztag, or Ganztag erweitert.
   - **Supplements (Zuschläge)** — check any that apply: **NdH**, **QM/MSS**, **Integration A**, **Integration B**. See [How funding works in Berlin](../../../explanation/how-funding-works-in-berlin/) if you're unsure what they mean.
5. Click **Save**.

The child now appears on the Children list with a calculated monthly funding amount.

## Notes

- Triple-check care type and supplements. They drive the funding amount and a wrong value will silently mismatch the next ISBJ Bescheid.
- If the parents bring an updated voucher (e.g. extension), don't edit the existing contract — create a new contract starting on the new effective date and let the old contract end naturally. This preserves the contract history.
- For the calculation chain (age × care_type × supplements → euros), see [How contract properties determine funding](../../../explanation/how-contract-properties-determine-funding/).
