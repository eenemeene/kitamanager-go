---
title: Record a child's departure
weight: 17
---

A child is leaving the Kita. You want to end their contract on the right day so the Bescheid reconciliation stays clean afterwards.

## Steps

1. Open the child's detail page.
2. In the **Contracts** section, click the active care contract.
3. Set **To** (called *Bis* in the German UI) to the last day the child is enrolled (e.g. their last attendance day, *not* the day they leave the building for the last time — the convention is "active up to and including this date").
4. Click **Save**.

The child stops contributing to staffing requirements and funding calculations from the day after the **To** date. They remain on the Children list (with no active contract) so historical reports stay correct.

## Notes

- Don't delete the child record. Deletion erases history; ending the contract preserves it for audits and reports.
- The next ISBJ Bescheid may still include the child for one more month if the Bezirks-Jugendamt hasn't processed the departure yet — that's a typical "extra in bill" mismatch and resolves itself the following month.
- If the child is moving up a section instead of leaving entirely, see [Move children between sections](../move-children-between-sections/).
- For the data-model rationale (why ending vs. deleting), see [How attendance is modeled](../../../explanation/how-attendance-is-modeled/) and the contract-history note at [Add a child](../add-a-child-and-issue-contract/).
