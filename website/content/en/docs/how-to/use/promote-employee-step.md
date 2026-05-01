---
title: Promote an employee step (Stufenaufstieg)
weight: 5
---

You want to record a TVöD-SuE step advancement so the employee's salary calculation reflects the higher step.

## Steps

1. Open the dashboard. The **Pending Step Promotions** widget lists every employee currently due for an advancement, with the current step, eligible step, the date the step becomes due, and the projected monthly cost delta.
2. Click the employee's name to open their detail page.
3. Scroll to the **Contracts** section. End the current contract by setting its **To** date to one day before the promotion date.
4. Click **Create Contract**. Set **From** to the promotion date, **Step** to the new step, and copy every other field from the old contract.
5. Click **Save**.

Refresh the dashboard. The widget should no longer list this employee.

## Notes

- The widget's projected cost delta includes both the salary increase and the employer contribution change — that's the total monthly impact.
- KitaManager uses the **From** date of the new contract to determine when the higher step takes effect, including in past months for retroactive promotions.
- The model — and how the date math works — is summarised in the explanation of [How the staffing key works](../../../explanation/what-the-staffing-key-means/) (the staffing model is independent but uses the same per-month-active contract logic).
