---
title: Screenshots
weight: 3
---

A walkthrough of the KitaManager user interface, showing the key screens you will use day to day. All screenshots show the included demo data ("Kita Sonnenschein").

---

## Login

{{< screenshot src="/images/screenshots/login.png" alt="Login page" caption="The login page — enter your email and password to access KitaManager." >}}

---

## Dashboard

The dashboard gives you an at-a-glance overview of your Kita: active employees, active children, staffing coverage, and alerts for children without vouchers, contract mismatches, upcoming enrollments, and section age limits.

{{< screenshot src="/images/screenshots/dashboard.png" alt="Dashboard" caption="The dashboard with summary cards and alerts for your organization." >}}

---

## Sections

Sections represent the groups within your Kita (e.g., Nest, Nestflüchter, Große). Drag and drop children between sections to reassign them.

{{< screenshot src="/images/screenshots/sections.png" alt="Sections" caption="The sections board with drag-and-drop for moving children between groups." >}}

---

## Employees

The employee list shows all staff members with their salary grade, weekly hours, and years of service. Hover over column headers for explanations.

{{< screenshot src="/images/screenshots/employees.png" alt="Employees list" caption="Employee overview with salary, grade, and staffing information." >}}

---

## Children

The children list shows every enrolled child with their calculated funding amount, FTE requirement, and billing difference.

{{< screenshot src="/images/screenshots/children.png" alt="Children list" caption="Children overview showing enrollment, funding amounts, and billing differences." >}}

---

## Statistics

The statistics hub provides access to all reports: financials, staffing, children, occupancy, budget, and forecast.

{{< screenshot src="/images/screenshots/statistics.png" alt="Statistics overview" caption="The statistics hub with summary cards for income, expenses, and balance." >}}

---

## Financial Overview

The financial overview shows income vs. expenses over time with a balance trend line.

{{< screenshot src="/images/screenshots/statistics-financials.png" alt="Financial overview" caption="Financial charts with income, expenses, and balance trend." >}}

The actual-vs-calculated funding comparison shows whether the government is paying correctly, with per-kita-year totals and deficit analysis.

{{< screenshot src="/images/screenshots/statistics-funding-comparison.png" alt="Actual vs calculated funding" caption="Actual government funding compared to calculated amounts." >}}

The cumulative balance chart tracks your running financial position. Red bars mark deficit periods.

{{< screenshot src="/images/screenshots/statistics-cumulative-balance.png" alt="Cumulative balance" caption="Cumulative balance with deficit markers." >}}

The budget overview table shows monthly income and expenses side by side.

{{< screenshot src="/images/screenshots/statistics-budget.png" alt="Budget overview" caption="Monthly budget breakdown with income, expenses, and balance." >}}

---

## Government Funding Bills

Upload ISBJ Excel files to compare government billing against calculated amounts. Navigate by Kita year, see a summary of matches and differences.

{{< screenshot src="/images/screenshots/government-funding-bills.png" alt="Government funding bills" caption="Funding bills filtered by Kita year with match/difference summary." >}}

Click on a bill to see the per-child comparison with status badges and mismatch indicators.

{{< screenshot src="/images/screenshots/government-funding-bill-detail.png" alt="Bill detail" caption="Per-child comparison showing billed vs. calculated amounts." >}}

---

## Child Billing History

View the complete billing history for an individual child across all uploaded bills.

{{< screenshot src="/images/screenshots/child-billing.png" alt="Child billing history" caption="Billing history for a child with running difference tracking." >}}

---

## Forecast

The forecast tool models what-if scenarios across the next Kita year. Three tabs let you optimize for a target balance, layer in hypothetical enrolments, or model hires and departures.

{{< screenshot src="/images/screenshots/forecast-optimize.png" alt="Forecast Optimize tab" caption="The Optimize tab finds the minimum number of children needed to reach a target balance." >}}

{{< screenshot src="/images/screenshots/forecast-children.png" alt="Forecast Children tab" caption="Add hypothetical children one at a time, or remove existing children to model an early departure." >}}

{{< screenshot src="/images/screenshots/forecast-employees.png" alt="Forecast Employees tab" caption="Model hires or departures the same way you model children." >}}

After clicking **Calculate Forecast**, the results panel projects the full year with your scenario applied — financials, cumulative balance, occupancy, and staffing.

{{< screenshot src="/images/screenshots/forecast-results.png" alt="Forecast results" caption="Projected income, costs, and balance for the Kita year with the scenario applied." >}}

---

## Settings & Two-Factor Authentication

The Settings page is where each user manages their password, two-factor authentication (TOTP and security keys), and the devices currently signed in.

{{< screenshot src="/images/screenshots/settings.png" alt="Settings page" caption="Settings — change password, manage 2FA, review active sessions." >}}

Enabling 2FA starts with a password confirmation, then a QR code you scan with an authenticator app (Google Authenticator, 1Password, Authy, etc.).

{{< screenshot src="/images/screenshots/settings-2fa-scan.png" alt="2FA QR code" caption="Scan the QR code (or enter the secret manually) to register your authenticator app." >}}

After confirming a one-time code, KitaManager generates single-use recovery codes — save them somewhere safe before clicking Done.

{{< screenshot src="/images/screenshots/settings-2fa-backup-codes.png" alt="2FA recovery codes" caption="Recovery codes are shown only once. Each code lets you sign in if you lose your authenticator." >}}

Once enrolment is complete, the 2FA card shows the active factors and lets you add a security key, regenerate recovery codes, or disable 2FA.

{{< screenshot src="/images/screenshots/settings-2fa-enabled.png" alt="2FA enabled" caption="Two-factor authentication active, with the option to add a security key or regenerate codes." >}}

---

## Audit Log

Admins can review every create, update, and delete made in their organization. Filter by date range or by action substring.

{{< screenshot src="/images/screenshots/audit-logs.png" alt="Audit log" caption="The org-scoped audit log shows who changed what and when." >}}

---

## Government Funding Rates

Configure the government funding rates for your state. Each entry maps child properties to a monthly funding amount.

{{< screenshot src="/images/screenshots/government-funding-rates.png" alt="Government funding rates" caption="Government funding configurations by state." >}}

The detail view shows periods, age ranges, and payment amounts per property.

{{< screenshot src="/images/screenshots/government-funding-rate-detail.png" alt="Funding rate detail" caption="Detailed funding rates with age ranges and payment amounts." >}}

---

## Pay Plans

Pay plans define salary grades and steps for your staff, typically based on the TVöD-SuE scale.

{{< screenshot src="/images/screenshots/payplan-detail.png" alt="Pay plan detail" caption="Pay plan showing salary grades, steps, and monthly amounts." >}}
