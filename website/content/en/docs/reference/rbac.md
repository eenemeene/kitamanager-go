---
title: RBAC
weight: 4
---

KitaManager uses Casbin-based, organisation-scoped RBAC. There are five roles. Every API request resolves the caller's role for the requested organisation, then asks Casbin whether that role can perform the action on the resource.

For *how to assign* roles, see [Manage users and roles](../../how-to/administer/). For the design rationale, see [Architecture: RBAC](../../explanation/architecture/#rbac-architecture).

## Roles

| Role | Scope | Description |
|---|---|---|
| `superadmin` | Global | Full system access across all organisations. Can create/delete organisations, manage funding configurations, view the global audit log. |
| `admin` | Organisation | Full access within assigned organisation(s). Can manage employees, children, contracts, sections, pay plans, and users. Cannot create/delete organisations or manage funding configurations. |
| `manager` | Organisation | Operational access within assigned organisation(s). Can manage employees, children, and contracts. Read-only on users, sections, pay plans. |
| `member` | Organisation | Read-only access within assigned organisation(s). Can view employees, children, contracts, sections, pay plans. Cannot modify anything. |
| `staff` | Organisation | Designed for educators tracking attendance. Read-only on children, child contracts, sections; full CRUD on attendance only. |

## Permission matrix

| Resource | Superadmin | Admin | Manager | Member | Staff |
|----------|-----------|-------|---------|--------|-------|
| Organisations | CRUD | Read/Update | Read | Read | Read |
| Employees | CRUD | CRUD | CRUD | Read | — |
| Children | CRUD | CRUD | CRUD | Read | Read |
| Contracts | CRUD | CRUD | CRUD | Read | Read (child only) |
| Attendance | CRUD | CRUD | CRUD | Read | CRUD |
| Sections | CRUD | CRUD | Read | Read | Read |
| Funding configurations | CRUD | — | — | — | — |
| Pay plans | CRUD | CRUD | Read | Read | — |
| Budget items | CRUD | CRUD | Read | Read | — |
| Statistics | Read | Read | Read | Read | — |
| Users | CRUD | CRUD | Read | — | — |
| ISBJ funding bills | Create / Read / Delete | Create / Read / Delete | Create / Read / Delete | — | — |

**Scope:** `superadmin` operates across all organisations. All other roles are scoped to the organisations they are members of. A user can be a member of multiple organisations with a different role in each (e.g. admin in Kita A, manager in Kita B).

## URL pattern for organisation-scoped resources

```
/api/v1/organizations/{orgId}/employees
/api/v1/organizations/{orgId}/children
/api/v1/organizations/{orgId}/sections
/api/v1/organizations/{orgId}/pay-plans
/api/v1/organizations/{orgId}/budget-items
/api/v1/organizations/{orgId}/government-funding-bills
/api/v1/organizations/{orgId}/statistics/...
/api/v1/organizations/{orgId}/audit-logs
```

Global resources (no `orgId` in the URL): `/api/v1/organizations`, `/api/v1/users`, `/api/v1/government-funding-rates`, `/api/v1/audit-logs` (superadmin), `/api/v1/me/...`.

## Authorization middleware

Handlers declare their requirements with one of:

```go
authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead)
authzMiddleware.RequireSuperAdmin()
```

The middleware extracts the user's session, resolves their role for the URL's `orgId` (if present), and asks Casbin. A failure returns `403 Forbidden`.
