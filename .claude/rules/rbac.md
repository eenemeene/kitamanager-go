---
paths:
  - "internal/handlers/**/*.go"
  - "internal/middleware/**/*.go"
  - "internal/rbac/**/*.go"
---

# RBAC

Casbin-based, organization-scoped multi-tenancy. The canonical reference is `website/content/en/docs/administration.md` (Role-Based Access Control section) — keep it in sync when adding roles or permissions.

## Roles

- `superadmin` — full system access across all organizations
- `admin` — full access within assigned organization(s)
- `manager` — operational access (employees, children, contracts)

(Additional roles like `member` and `staff` are documented in the canonical reference.)

## URL pattern for organization-scoped resources

```
/api/v1/organizations/{orgId}/employees
/api/v1/organizations/{orgId}/children
```

## Authorization middleware

```go
authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead)
authzMiddleware.RequireSuperAdmin()
```
