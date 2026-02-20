---
title: Screenshots
weight: 3
---

A walkthrough of the KitaManager Go user interface, showing the key screens you will use day to day.

---

## Login

The login screen is the entry point to KitaManager. Users authenticate with their email address and password. After successful login, a JWT token is issued that keeps the session active.

{{< screenshot src="/images/screenshots/login.png" alt="Login page" caption="The login page — enter your email and password to access the system." >}}

---

## Dashboard

Once logged in, the dashboard gives you an at-a-glance overview of your Kita. The top row shows summary cards for the total number of organizations, employees, children, and users. Below that, **Quick Stats** shows details for the currently selected organization.

The left sidebar is your main navigation — it is divided into system-wide items (Dashboard, Organizations, Government Fundings) and organization-scoped items (Users, Groups, Employees, Children, Statistics, Pay Plans). You can switch between organizations using the dropdown in the sidebar.

{{< screenshot src="/images/screenshots/dashboard.png" alt="Dashboard" caption="The dashboard with summary cards and organization-scoped quick stats." >}}

---

## Organizations

The organizations page lists all Kita facilities you have access to. Each row shows the organization name, its German state (Bundesland), and whether it is currently active. From here you can create new organizations or edit existing ones.

If you manage multiple Kitas, this is where you get the full picture across all your facilities.

{{< screenshot src="/images/screenshots/organizations.png" alt="Organizations list" caption="Organizations overview — manage multiple Kita facilities from one place." >}}

---

## Employees

The employee list shows all staff members for the selected organization. You can see each employee's name, gender, birthdate, age, current position, pay grade and step, and weekly hours at a glance. The action buttons on the right let you view contract history, view details, edit the employee record, or remove an employee.

New employees can be added using the **+ New Employee** button in the top right corner.

{{< screenshot src="/images/screenshots/employees.png" alt="Employees list" caption="Employee overview with personal details, position, grade, and weekly hours." >}}

---

## Children

The children list shows every enrolled child in the selected organization. Each row displays the child's name, gender, birthdate, age, current contract status, care properties (like halbtag, ganztag, ndh, or integration), and the **automatically calculated monthly funding amount** based on the active government funding configuration.

This is the screen where Kita administrators spend most of their time — it gives a complete picture of enrollment and funding at a glance.

{{< screenshot src="/images/screenshots/children.png" alt="Children list" caption="Children overview showing enrollment status, care properties, and calculated funding amounts." >}}

---

## Government Funding

The government funding page lets administrators configure the state-level funding rules that drive the automatic funding calculations. Each entry represents a funding configuration for a specific German state (e.g., "Berlin Kita-Foerderung"). Within each configuration, you can define time periods and property-based funding amounts.

When a child's contract properties match a funding entry, the corresponding monthly amount is automatically displayed in the children list.

{{< screenshot src="/images/screenshots/government-funding-rates.png" alt="Government funding" caption="Government funding configurations — define state-level rules for automatic funding calculations." >}}
