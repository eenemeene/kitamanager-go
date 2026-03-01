---
title: User Guide
weight: 4
---

This guide walks you through the everyday tasks you can perform in KitaManager. It is written for daycare teachers and administrators who use the application on a daily basis.

## Logging In

1. Open your browser and navigate to the KitaManager URL provided by your administrator.
2. Enter your email address and password.
3. Click **Login**.
4. You will be taken to the dashboard, which shows an overview of your organization.

If you have forgotten your password, contact your administrator to reset it.

{{< screenshot src="/images/screenshots/en/login.png" alt="Login page" caption="The KitaManager login page." >}}

## Navigating the Interface

The application has the following navigation elements:

- **Sidebar** -- The main menu on the left side of the screen. It contains links to all major sections such as Employees, Children, Sections, Statistics, and more. On mobile devices, tap the menu icon to open the sidebar.
- **Organization selector** -- Located in the sidebar. If you have access to multiple organizations, you can switch between them here.
- **Breadcrumbs** -- Displayed at the top of each page, showing your current location in the application. Click any breadcrumb to navigate back to that level.
- **Dark mode toggle** -- Switch between light and dark color themes using the toggle in the header.
- **Language switcher** -- Switch the interface between English and German (EN/DE) using the language selector in the header.

{{< screenshot src="/images/screenshots/en/dashboard.png" alt="Dashboard" caption="The dashboard provides an overview of your organization." >}}

## Switching Organizations

If your account has access to more than one organization:

1. Look for the organization selector in the sidebar.
2. Select the organization you want to work with.
3. All data on the screen will update to reflect the selected organization.

Any actions you take (creating employees, enrolling children, etc.) will apply to the currently selected organization.

## Managing Sections

Sections represent groups within your daycare, such as "Schmetterlinge" or "Sonnenkinder". They help you organize children and employees into their respective groups.

### Viewing Sections

1. Click **Sections** in the sidebar.
2. You will see a list of all sections in your organization.

{{< screenshot src="/images/screenshots/en/sections.png" alt="Sections list" caption="The sections page showing all groups in your organization." >}}

### Creating a Section

1. Navigate to **Sections**.
2. Click the **Create** button.
3. Enter the section name.
4. Click **Save**.

### Editing or Deleting a Section

1. Navigate to **Sections**.
2. Find the section you want to modify.
3. Click on the section to open its detail view.
4. To edit, update the fields and click **Save**.
5. To delete, click the **Delete** button and confirm the action.

## Managing Employees

### Viewing Employees

1. Click **Employees** in the sidebar.
2. You will see a list of all employees in your organization.

{{< screenshot src="/images/screenshots/en/employees.png" alt="Employees list" caption="The employees page listing all staff members." >}}

### Creating an Employee

1. Navigate to **Employees**.
2. Click the **Create** button.
3. Fill in the required fields:
   - First name
   - Last name
   - Gender
   - Birthdate
4. Click **Save**.

### Editing or Deleting an Employee

1. Navigate to **Employees** and click on the employee you want to modify.
2. To edit, update the fields and click **Save**.
3. To delete, click the **Delete** button and confirm the action.

### Creating an Employment Contract

Each employee can have one or more employment contracts that define their working conditions and salary.

1. Navigate to the employee's detail page.
2. In the contracts section, click **Create Contract**.
3. Fill in the contract details:
   - **From** -- Start date of the contract
   - **To** -- End date of the contract (leave empty for open-ended contracts)
   - **Staff category** -- The employee's role category
   - **Grade** -- Salary grade
   - **Step** -- Current salary step within the grade
   - **Weekly hours** -- Number of hours worked per week
   - **Pay plan** -- The applicable pay plan
   - **Section** -- The section the employee is assigned to
4. Click **Save**.

The current contract is displayed on the employee's detail page.

{{< screenshot src="/images/screenshots/en/employee-contract-create.png" alt="Employee contract creation dialog" caption="The dialog for creating a new employment contract." >}}

{{< screenshot src="/images/screenshots/en/employee-contracts.png" alt="Employee contracts" caption="Employment contracts for a staff member." >}}

### Step Promotions

KitaManager tracks which employees are eligible for a promotion to the next salary step based on how long they have been in their current step.

1. Navigate to **Employees**.
2. Look for indicators showing step promotion eligibility.
3. Review the list to identify employees due for a salary step increase.

## Enrolling Children

### Viewing Children

1. Click **Children** in the sidebar.
2. You will see a list of all enrolled children along with their funding amounts.

{{< screenshot src="/images/screenshots/en/children.png" alt="Children list" caption="The children page showing all enrolled children and their funding amounts." >}}

### Creating a Child Record

1. Navigate to **Children**.
2. Click the **Create** button.
3. Fill in the required fields:
   - First name
   - Last name
   - Gender
   - Birthdate
4. Click **Save**.

### Editing or Deleting a Child Record

1. Navigate to **Children** and click on the child you want to modify.
2. To edit, update the fields and click **Save**.
3. To delete, click the **Delete** button and confirm the action.

### Creating a Care Contract

Care contracts define how a child is enrolled and determine the government funding amount.

1. Navigate to the child's detail page.
2. In the contracts section, click **Create Contract**.
3. Fill in the contract details:
   - **From** -- Start date of the contract
   - **To** -- End date of the contract
   - **Voucher number** -- The government-issued voucher number
   - **Section** -- The section the child is assigned to
4. Set the contract properties:
   - **Care type** -- Choose between Halbtag (half-day), Ganztag (full-day), or Teilzeit (part-time)
   - **Supplements** -- Select any applicable supplements:
     - NDH (non-German-speaking household)
     - MSS
     - Integration A
     - Integration B
5. Click **Save**.

The contract properties determine the government funding amount that KitaManager calculates for each child.

{{< screenshot src="/images/screenshots/en/child-contract-create.png" alt="Child contract creation dialog" caption="The dialog for creating a new care contract with contract properties." >}}

{{< screenshot src="/images/screenshots/en/child-contracts.png" alt="Child contracts" caption="Care contracts for an enrolled child." >}}

## Daily Attendance Tracking

Staff members can track the daily attendance of children.

### Marking Attendance

1. Click **Attendance** in the sidebar.
2. You will see a weekly grid view showing all children who have active care contracts.
3. For each child, mark them as **present** or **absent** for each day of the week.
4. Your changes are saved automatically.

{{< screenshot src="/images/screenshots/en/attendance.png" alt="Attendance tracking" caption="The weekly attendance grid for tracking children's daily presence." >}}

### Viewing Attendance Summary

- The attendance page shows an organization-wide summary of attendance for the selected week.
- To view the attendance history for a specific child, navigate to the child's detail page and look for the attendance section.

## Budget Management

Budget management allows you to track income and expenses for your organization.

### Creating a Budget Item

Budget items represent categories of income or expenses (for example, "Office Supplies" or "Parent Contributions").

1. Click **Budget** in the sidebar.
2. Click the **Create** button.
3. Enter the budget item name and the total budget amount.
4. Click **Save**.

{{< screenshot src="/images/screenshots/en/budget-items.png" alt="Budget items" caption="The budget overview showing all budget items for your organization." >}}

### Adding Entries to a Budget Item

Each budget item can have multiple entries that represent individual transactions.

1. Navigate to the budget item's detail page.
2. Click **Add Entry**.
3. Fill in the details:
   - **From** -- Start date
   - **To** -- End date
   - **Amount** -- The monetary amount
4. Click **Save**.

Use entries to track actual spending or income against your budgeted amounts.

{{< screenshot src="/images/screenshots/en/budget-item-detail.png" alt="Budget item detail" caption="A budget item with its individual entries." >}}

## Viewing Statistics

KitaManager provides several reports to help you understand your organization's data.

1. Click **Statistics** in the sidebar.
2. Select the report you want to view:
   - **Staffing hours** -- Total working hours broken down by staff category
   - **Financial overview** -- Salary costs and employer contributions
   - **Occupancy** -- Number of enrolled children compared to capacity
   - **Age distribution** -- Children grouped by age
   - **Contract properties** -- Distribution of care types (half-day, full-day, part-time)
   - **Funding** -- Calculated government funding totals
3. Use the **date range** filter to adjust the time period.
4. Use the **section** filter to narrow results to a specific group.
5. To print a report, click the **Print** button.

{{< screenshot src="/images/screenshots/en/statistics.png" alt="Statistics overview" caption="The statistics overview page." >}}

{{< screenshot src="/images/screenshots/en/statistics-staffing.png" alt="Staffing hours chart" caption="Staffing hours report showing required vs. available hours over time." >}}

{{< screenshot src="/images/screenshots/en/statistics-financials.png" alt="Financial overview charts" caption="Financial overview with income, expenses, and funding breakdown." >}}

{{< screenshot src="/images/screenshots/en/statistics-children.png" alt="Children statistics" caption="Children statistics with age distribution and contract properties." >}}

{{< screenshot src="/images/screenshots/en/statistics-occupancy.png" alt="Occupancy table" caption="Occupancy report showing enrolled children compared to capacity." >}}

## Government Funding Bills

You can compare government funding bills against the amounts KitaManager calculates to identify discrepancies.

### Uploading a Bill

1. Navigate to the **ISBJ Bills** section.
2. Click **Upload** and select the ISBJ bill file from your computer.
3. The uploaded bill will appear in the list.

{{< screenshot src="/images/screenshots/en/government-funding-bills.png" alt="Government funding bills" caption="Uploaded government funding bills for comparison with calculated amounts." >}}

### Reviewing Discrepancies

1. Open an uploaded bill.
2. KitaManager displays the calculated funding amounts alongside the billed amounts.
3. Review the comparison to find any differences between what was billed and what KitaManager calculated based on the children's contracts.

## Importing Data

You can import data from YAML files to quickly populate your organization.

1. Navigate to the relevant page (Children, Employees, or the applicable settings page).
2. Click the **Import** button.
3. Select the YAML file from your computer.
4. Review and confirm the import.

The following data types can be imported:

| Data Type | Format |
|-----------|--------|
| Children | YAML |
| Employees | YAML |
| Pay plans | YAML |
| Government funding rates | YAML |

## Exporting Data

You can export data for backup or use in other applications.

1. Navigate to the relevant page (Children, Employees, or Pay Plans).
2. Click the **Export** button.
3. Select the desired format.

The following export formats are available:

| Data Type | Formats |
|-----------|---------|
| Children | Excel, YAML |
| Employees | Excel, YAML |
| Pay plans | YAML |

Exported Excel files can be opened in spreadsheet applications such as Microsoft Excel or LibreOffice Calc.
