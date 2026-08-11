---
title: Investigate a bill discrepancy
weight: 7
---

You uploaded an ISBJ bill and the comparison shows mismatches. You want to know which child to look at and what to fix.

## The three mismatch categories

When the bill and KitaManager don't agree, every non-matching child falls into exactly one of:

- **Missing from bill** — KitaManager expects funding for this child, the Bescheid doesn't include them. Most often: the Gutscheinnummer in your record is wrong, or the voucher hasn't been processed by the Bezirks-Jugendamt yet.
- **Extra in bill** — the Bescheid pays for a child who doesn't exist in KitaManager (or whose contract has ended). Most often: a child left and the district hasn't updated their record, or the child was never created in KitaManager.
- **Different rates** — both sides agree the child should be funded but the amounts differ. Most often: the contract properties (care type, supplements) don't match what the district has on file.

## What to do per category

### Missing from bill

1. Open the child's detail page in KitaManager.
2. Confirm the Gutscheinnummer matches the paper voucher. Typos here cause >90% of "missing" cases.
3. If the voucher is recent, give the district a billing cycle to catch up.
4. If the voucher is correct and not recent, contact the Bezirks-Jugendamt: they may not have processed the enrolment.

### Extra in bill

1. Note the Gutscheinnummer from the bill row and search for it in the Children list.
2. If the child exists with a contract that has an end date in the past, the district hasn't recorded the departure. Notify them.
3. If the child doesn't exist in KitaManager at all, decide: should they exist (in which case create the record), or is the bill referencing a child you never had (in which case dispute the line with the district)?

### Different rates

1. Open the child's detail page → Contracts.
2. Compare each contract field with the paper voucher: care type, NdH, QM/MSS, Integrationsstatus.
3. The most common drift: NdH set on one side and not the other. NdH is updated by the district independently of voucher renewal.
4. Fix the contract in KitaManager (or contact the district to fix theirs, depending on which side is wrong).

## Other failure modes

| Symptom | Likely cause | Where to look |
|---|---|---|
| The file is rejected on upload | Excel layout drift or wrong file type | parser column map in `internal/isbj/parse.go`; verify file is the actual Bescheid |
| "No bills found for this month" | Upload created a different period than expected | bill period dates on the Funding Bills page |
| Many "missing from bill" entries | Voucher numbers don't match | child detail pages — see steps above |
| Many "extra in bill" entries | Children left, Senate hasn't processed the departure | Bezirks-Jugendamt |
| Persistent total drift across many children | Funding rates outdated | [Update government funding rates](../../operate/update-government-funding-rates/) |
| Total matches but property breakdown differs | Supplement out of sync | child contract supplements vs paper voucher |

## Notes

- After fixing, re-upload the bill. The comparison will refresh and the row should match.
- For the calculation, see [How contract properties determine funding](../../../explanation/how-contract-properties-determine-funding/) and [How funding works in Berlin](../../../explanation/how-funding-works-in-berlin/).
- For the sequence used when reading the file, see [The ISBJ reconciliation flow](../../../explanation/the-isbj-reconciliation-flow/).
