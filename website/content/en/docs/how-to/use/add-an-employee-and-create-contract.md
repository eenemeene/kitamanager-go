---
title: Add an employee and create an employment contract
weight: 4
---

You want to record a new staff member and define their working conditions.

## Steps

### Create the employee record

1. Click **Employees** in the sidebar.
2. Click **Create**.
3. Fill in **First name**, **Last name**, **Gender**, **Birthdate**.
4. Click **Save**.

### Create the employment contract

1. Open the employee's detail page.
2. Scroll to the **Contracts** section and click **Create Contract**.
3. Fill in:
   - **From** — contract start date.
   - **To** — end date (leave empty for permanent contracts).
   - **Staff category (Personalkategorie)** — Fachkraft, Hilfskraft, Leitung, etc.
   - **Grade (Entgeltgruppe)** — e.g. `S 8a`.
   - **Step (Stufe)** — current experience step (1–6 for TVöD-SuE).
   - **Weekly hours (Wochenstunden)** — hours per week.
   - **Pay plan (Entgelttabelle)** — TVöD-SuE 2024, Minijob, etc.
   - **Section** — which group the employee is assigned to.
4. Click **Save**.

The employee now contributes to the staffing-hours calculation for their section.

## Notes

- When the employee's conditions change (more hours, different section, step promotion), don't edit the existing contract — create a new one starting on the change date. The old contract documents the prior period; KitaManager uses both for historical reporting.
- A wrong grade/step/hours combination silently miscomputes salary cost and staffing coverage. Double-check before saving.
- For the salary calculation, see [Pay plans (TVöD-SuE)](../../administer/update-pay-plan/) on the admin side.
