---
title: Update a child's care contract on a change
weight: 19
---

A child's circumstances change: care hours go up (Halbtag → Ganztag), a supplement starts or stops (NdH applies after the family communication language changes, Integrationsstatus A or B is granted by the Bezirks-Jugendamt), or the parents bring a new Kita-Gutschein. You want to record the change so the next ISBJ Bescheid still matches what KitaManager calculates.

This recipe covers care-type and supplement changes. For the other update workflows the recipes are dedicated:

- New voucher number → [Assign a Kita-Gutschein number](../assign-a-voucher/)
- Section change → [Move children between sections](../move-children-between-sections/)
- Contract end (child leaves) → [Record a child's departure](../record-a-childs-departure/)

## The rule: extend history, don't edit

Care type and supplements drive the calculated monthly funding amount (see [How contract properties determine funding](../../../explanation/how-contract-properties-determine-funding/)). When you edit them on the existing contract, **historical months silently recompute against the new values** — bills you've already reconciled stop matching what KitaManager would calculate today, and the audit log only shows the "after" state.

Instead: a real change has an effective date. End the current contract on the day before. Create a new contract from the effective date with the new properties.

The one exception is **correcting an input error** — you typed `halbtag` but meant `ganztag` from day one. Edit the contract; the timeline already reflects what should have been there.

## Steps — care-type or supplement change going forward

{{< screenshot src="/images/screenshots/child-contracts.png" alt="A child's contract history" caption="The child's contract history showing care type and supplements as badges." >}}

1. Open the child from the **Children** list and click the **history** icon to open their contract history.
2. Find the **active** contract (status badge: *active*). Click the **pencil** to edit.
3. Set **To** to the day before the change takes effect (e.g. Integrationsstatus A is granted from 1 February → To = 31 January). Click **Save**. The **To** date must be today or later — for an effective date in the past, see [Backdated changes](#backdated-changes).
4. Back on the contract history, click **New contract**.
5. Set **From** to the effective date. Pick the same **Section** (unless the section is also changing — then see [Move children between sections](../move-children-between-sections/)).
6. Set the contract **Properties**:
   - **Care type (Betreuungsart)** — pick the right one: Halbtag, Teilzeit, Ganztag, Ganztag erweitert.
   - **Supplements (Zuschläge)** — check every applicable supplement: NdH, QM/MSS, Integration A, Integration B. Remove any that no longer apply.
7. Click **Save**.

{{< screenshot src="/images/screenshots/child-contract-create.png" alt="New child contract dialog" caption="Pick care type and check every applicable supplement." >}}

The child's calculated monthly funding amount updates immediately — verify the new amount on the Children list before moving on.

## Common change scenarios

| Trigger | What to change |
|---|---|
| Care hours change (Halbtag → Ganztag etc.) | **Care type** in properties |
| Family changes communication language to non-German | Add **NdH** supplement |
| Kita is reclassified into a QM/MSS area | Add **QM/MSS** supplement |
| Bezirks-Jugendamt grants Eingliederungshilfe | Add **Integration A** or **Integration B** |
| Supplement expires or is revoked | Remove that supplement |
| Voucher extension or new voucher number | See [Assign a Kita-Gutschein](../assign-a-voucher/) — same contract, no end-and-create needed |

## Backdated changes

When the effective date is in the past (the Bescheid arrives in August, the change applies from 1 February), the interface currently **cannot** record it.

Here's why: for a contract that started before today, saving an edit makes KitaManager create the follow-on contract **starting today**. A **To** date in the past would give that contract an end before its start, which KitaManager rejects ("from date must be before or equal to to date"). The reverse route — creating the backdated contract first — is rejected too, because it overlaps the running contract.

What you can do instead:

1. Record the change **from today**, as described above. From now on KitaManager calculates correctly.
2. Expect the months between the effective date and today to keep using the old values and to show up as discrepancies during reconciliation. [Investigate a bill discrepancy](../investigate-a-bill-discrepancy/) helps when checking those.

This is a known limitation, not a mistake on your part. If you regularly receive backdated Bescheide, this is the gap worth closing in the product.

## Fix a wrong date (no new contract needed)

If only the start or end **date** is wrong (you wrote March 1 when the contract really started March 15), use one of:

- **Edit dialog** — open the contract, change **From** / **To**, save.
- **Timeline view** — switch to the **Timeline** tab on the contracts page and drag the contract boundary.

## Notes

- A wrong care-type or missing supplement silently mismatches the next ISBJ Bescheid. The mismatch can be hundreds of euros per child per month — fix it as soon as you learn about the change.
- Don't delete the old contract. Ending it preserves the historical record; deletion erases it from staffing, occupancy, and funding reports.
- For the lookup chain (age × care_type × supplements → euros), see [How contract properties determine funding](../../../explanation/how-contract-properties-determine-funding/).
- The audit log records every contract create / edit / delete with old → new values. Admins can review it via [Review the audit log](../../administer/review-audit-log/).
