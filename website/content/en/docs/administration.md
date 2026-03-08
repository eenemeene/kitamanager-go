---
title: Administration Guide
weight: 5
---

This guide covers administrative tasks in KitaManager: managing organizations, users, roles, funding configuration, and pay plans. You need admin or superadmin access to perform most of the actions described here.

## Managing Organizations

Organizations represent individual daycare centers (Kitas). Each organization is a separate data space -- children, employees, contracts, and other records belong to exactly one organization.

### Creating an Organization

Only **superadmins** can create organizations. When creating an organization, you must provide:

- **Name** -- the display name for the daycare center (e.g., "Kita Sonnenschein")
- **State (Bundesland)** -- the German federal state where the organization is located

The state determines which government funding rules apply. Supported states include Berlin, Brandenburg, Bayern, and other German federal states.

### Updating an Organization

Admins and superadmins can update organization details such as the name and state.

### Deleting an Organization

Only **superadmins** can delete organizations. Deleting an organization removes all associated data (children, employees, contracts, etc.).

## User Management

User management is available to admins and superadmins.

### Creating a User

When creating a user, provide:

- **Name** -- the user's display name
- **Email** -- used for login, must be unique
- **Password** -- must meet minimum length requirements
- **Active** -- whether the account is enabled

### Listing Users

Admins can view all users. The user list supports pagination and shows each user's name, email, and active status.

### Editing and Deleting Users

Admins can update user details (name, email, active status) and delete user accounts. Deleting a user removes their role assignments and access.

### Resetting Passwords

Admins can reset a user's password through the user management interface.

### Superadmin Status

Only existing superadmins can grant or revoke superadmin status on other users. This is done through a dedicated toggle in the user management interface.

## Role-Based Access Control

KitaManager uses five roles to control access. Each role has a defined set of permissions that determine what a user can do.

### Roles Overview

- **Superadmin** -- global system administrator with full access across all organizations. Can create and delete organizations, manage funding configurations, and perform all operations.
- **Admin** -- full control within assigned organizations. Can manage employees, children, contracts, sections, pay plans, and users. Cannot create or delete organizations or manage funding configurations.
- **Manager** -- handles daily operational tasks within assigned organizations. Can manage employees, children, and contracts. Has read-only access to users, sections, and pay plans.
- **Member** -- read-only access within assigned organizations. Can view employees, children, contracts, sections, and pay plans but cannot modify anything.
- **Staff** -- designed for teachers and assistants who need to track attendance. Can view children, child contracts, and sections. Has full create/read/update/delete access to attendance records only.

### Permission Matrix

| Resource | Superadmin | Admin | Manager | Member | Staff |
|----------|-----------|-------|---------|--------|-------|
| Organizations | CRUD | Read/Update | Read | Read | Read |
| Employees | CRUD | CRUD | CRUD | Read | -- |
| Children | CRUD | CRUD | CRUD | Read | Read |
| Contracts | CRUD | CRUD | CRUD | Read | Read (child only) |
| Attendance | CRUD | CRUD | CRUD | Read | CRUD |
| Sections | CRUD | CRUD | Read | Read | Read |
| Funding Config | CRUD | -- | -- | -- | -- |
| Pay Plans | CRUD | CRUD | Read | Read | -- |
| Budget | CRUD | CRUD | Read | Read | -- |
| Statistics | Read | Read | Read | Read | -- |
| Users | CRUD | CRUD | Read | -- | -- |
| Gov. Funding Bills | Create/Read/Delete | Create/Read/Delete | Create/Read/Delete | -- | -- |

**Scope:** Superadmins operate across all organizations. All other roles are scoped to their assigned organizations only.

## Organization Membership

Users are assigned to organizations with a specific role. This determines what they can access and where.

### Key Concepts

- A user can belong to **multiple organizations** with different roles in each. For example, a user might be an admin in one Kita and a manager in another.
- Role assignments are managed through the user management interface. Admins can add or remove organization memberships for users within their own organizations.
- Superadmins can manage memberships across all organizations.

### Assigning a Role

To assign a user to an organization, select the user in the user management interface, choose the target organization, and assign the desired role (admin, manager, member, or staff).

### Removing a Membership

Removing a user's membership from an organization revokes their access to that organization's data. The user account itself is not deleted.

## Configuring Government Funding Rates

Government funding configuration is a **superadmin-only** operation. It defines the funding rates that government agencies pay for childcare based on the state's regulations.

### Structure

A funding configuration consists of:

1. **Funding Configuration** -- a top-level entry with a name and the associated state (Bundesland)
2. **Time Periods** -- date ranges (from/to) within a configuration, each with a full-time weekly hours value
3. **Properties** -- individual funding rate entries within a period

### Properties

Each property defines a specific funding rate with the following fields:

| Field | Description | Example |
|-------|-------------|---------|
| Key | Category identifier | `care_type` |
| Value | Specific value within the category | `ganztag` |
| Label | Human-readable description | "Full-day care" |
| Payment | Amount in cents | `166847` (= 1,668.47 EUR) |
| Min Age | Minimum child age (months) | `0` |
| Max Age | Maximum child age (months) | `36` |
| Apply to All | Whether this rate applies universally | `true` / `false` |

{{% callout type="info" %}}
All monetary values are stored as integers in cents to avoid floating-point precision errors. For example, 1,668.47 EUR is stored as `166847`.
{{% /callout %}}

### Importing Funding Rates

Funding rates can be imported from YAML files. This is useful for bulk-loading official government rate tables. The YAML format defines the full configuration including periods and properties.

## Configuring Pay Plans

Pay plans define salary structures for employees, typically following collective bargaining agreements such as TVoD-SuE.

### Structure

A pay plan consists of:

1. **Pay Plan** -- a named plan (e.g., "TVoD-SuE") belonging to an organization
2. **Periods** -- date ranges with associated weekly hours and employer contribution rate
3. **Entries** -- individual salary entries within a period

### Periods

Each period defines:

| Field | Description | Example |
|-------|-------------|---------|
| From | Start date | 2025-01-01 |
| To | End date | 2025-12-31 |
| Weekly Hours | Standard weekly working hours | 39.0 |
| Employer Contribution Rate | Rate in hundredths of a percent | `2050` (= 20.50%) |

### Entries

Each entry within a period defines:

| Field | Description | Example |
|-------|-------------|---------|
| Grade | Pay grade | `S8a` |
| Step | Experience step (1--6) | `3` |
| Monthly Amount | Salary in cents | `385000` (= 3,850.00 EUR) |
| Minimum Years | Minimum years of experience for this step | `5` |

### Import and Export

Pay plans can be imported from and exported to YAML files. This simplifies setting up standard pay structures and sharing them across organizations.

## Audit Logging

All create, update, and delete operations in KitaManager are recorded in the audit log. This supports compliance requirements and helps track who changed what and when.

### Logged Information

Each audit log entry contains:

| Field | Description |
|-------|-------------|
| Actor | The user who performed the action |
| Resource Type | The type of resource affected (e.g., employee, child, contract) |
| Resource ID | The database ID of the affected resource |
| Resource Name | A human-readable name for the affected resource |
| IP Address | The IP address from which the action was performed |
| Timestamp | When the action occurred |

Audit logs are read-only and cannot be modified or deleted.

## Seed Data

Development and testing environments can be populated with sample data to help with setup and testing.

### What Gets Seeded

- A sample organization ("Kita Sonnenschein")
- Test children with contracts
- Sample employees
- Berlin state government funding configuration

### Running the Seed

Use the Makefile target to seed the database:

```bash
make seed
```

Alternatively, the seeding API endpoint can be called directly in development mode.

{{% callout type="warning" %}}
Seed data is intended for development and testing only. Do not run seeding on production databases.
{{% /callout %}}

## Next Steps

- [Getting Started](../getting-started/) -- Set up the application
- [Architecture Overview](../architecture/) -- Understand the system design
- [API Reference](../api/) -- Explore the REST API
