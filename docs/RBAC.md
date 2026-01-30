# Role-Based Access Control (RBAC)

KitaManager uses a hybrid RBAC system combining database-stored role assignments with [Casbin](https://casbin.org/) policy evaluation.

## Architecture Overview

The RBAC system has two components:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Authorization Flow                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Request comes in                                            │
│          │                                                      │
│          ▼                                                      │
│  2. AuthorizationMiddleware extracts userID and orgID           │
│          │                                                      │
│          ▼                                                      │
│  3. PermissionService.CheckPermission(userID, orgID, ...)       │
│          │                                                      │
│          ├──► Check if superadmin (User.IsSuperAdmin in DB)     │
│          │         │                                            │
│          │         └──► If yes: ALLOW                           │
│          │                                                      │
│          ├──► Get role from DB (UserGroup table)                │
│          │         │                                            │
│          │         └──► If no role: DENY                        │
│          │                                                      │
│          └──► Check role permission via Casbin                  │
│                    │                                            │
│                    └──► enforcer.Enforce(role, "*", resource,   │
│                              action)                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Hybrid Approach?

| Storage | What | Why |
|---------|------|-----|
| **Database** (UserGroup table) | User → Role → Organization assignments | Queryable, auditable, integrates with GORM transactions |
| **Casbin** (casbin_rule table) | Role → Permission mappings | Optimized for policy evaluation, declarative definitions |

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `PermissionService` | `internal/rbac/permission.go` | Production authorization checks |
| `Enforcer` | `internal/rbac/rbac.go` | Casbin wrapper, policy seeding |
| `AuthorizationMiddleware` | `internal/middleware/authorization.go` | HTTP middleware |
| `UserGroup` model | `internal/models/user_group.go` | Role assignments |

## Roles

| Role | Description |
|------|-------------|
| `superadmin` | Full system access. Can create/delete organizations and access all resources across all organizations. Stored as `User.IsSuperAdmin` boolean. |
| `admin` | Full access within assigned organization(s). Cannot create/delete organizations. |
| `manager` | Operational access. Can manage employees, children, and contracts. Read-only access to users and groups. |
| `member` | Read-only access to employees, children, and contracts within their organization. |

## Resources

| Resource | Description |
|----------|-------------|
| `organizations` | Kita/daycare centers |
| `employees` | Staff members |
| `children` | Enrolled children |
| `employee_contracts` | Employment contracts |
| `child_contracts` | Enrollment contracts |
| `users` | System users |
| `groups` | User groups |
| `sections` | Organizational sections |
| `fundings` | Government funding definitions |
| `payplans` | Pay plan definitions |

## Actions

| Action | Description |
|--------|-------------|
| `create` | Create new records |
| `read` | View records |
| `update` | Modify existing records |
| `delete` | Remove records |

## Permission Matrix

### Superadmin

Full CRUD access to all resources in all organizations.

### Admin (within assigned organization)

| Resource | Create | Read | Update | Delete |
|----------|--------|------|--------|--------|
| organizations | ❌ | ✅ | ✅ | ❌ |
| employees | ✅ | ✅ | ✅ | ✅ |
| children | ✅ | ✅ | ✅ | ✅ |
| employee_contracts | ✅ | ✅ | ✅ | ✅ |
| child_contracts | ✅ | ✅ | ✅ | ✅ |
| users | ✅ | ✅ | ✅ | ✅ |
| groups | ✅ | ✅ | ✅ | ✅ |
| sections | ✅ | ✅ | ✅ | ✅ |
| payplans | ✅ | ✅ | ✅ | ✅ |

### Manager (within assigned organization)

| Resource | Create | Read | Update | Delete |
|----------|--------|------|--------|--------|
| organizations | ❌ | ✅ | ❌ | ❌ |
| employees | ✅ | ✅ | ✅ | ✅ |
| children | ✅ | ✅ | ✅ | ✅ |
| employee_contracts | ✅ | ✅ | ✅ | ✅ |
| child_contracts | ✅ | ✅ | ✅ | ✅ |
| users | ❌ | ✅ | ❌ | ❌ |
| groups | ❌ | ✅ | ❌ | ❌ |
| sections | ❌ | ✅ | ❌ | ❌ |
| payplans | ❌ | ✅ | ❌ | ❌ |

### Member (within assigned organization)

| Resource | Create | Read | Update | Delete |
|----------|--------|------|--------|--------|
| organizations | ❌ | ✅ | ❌ | ❌ |
| employees | ❌ | ✅ | ❌ | ❌ |
| children | ❌ | ✅ | ❌ | ❌ |
| employee_contracts | ❌ | ✅ | ❌ | ❌ |
| child_contracts | ❌ | ✅ | ❌ | ❌ |
| sections | ❌ | ✅ | ❌ | ❌ |
| payplans | ❌ | ✅ | ❌ | ❌ |

## API Endpoints

### Organization-Scoped Endpoints

Resources that belong to an organization use the URL pattern:
```
/api/v1/organizations/{orgId}/employees
/api/v1/organizations/{orgId}/children
/api/v1/organizations/{orgId}/sections
```

The middleware extracts `orgId` from the URL and checks permissions for that organization.

### Global Endpoints

Some resources are not organization-scoped in the URL but still require org-based permissions:
```
/api/v1/users
/api/v1/groups
```

These use `RequireGlobalPermission` which checks if the user has the required permission in ANY of their assigned organizations.

### Superadmin-Only Endpoints

```
POST   /api/v1/organizations      # Create organization
DELETE /api/v1/organizations/{id} # Delete organization
POST   /api/v1/government-fundings # Create funding definition
```

## Data Model

### User → Role Assignment (Database)

```sql
-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    is_super_admin BOOLEAN DEFAULT FALSE,  -- Superadmin flag
    ...
);

-- user_groups table (role assignments)
CREATE TABLE user_groups (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    group_id INTEGER REFERENCES groups(id),  -- Group belongs to an org
    role VARCHAR(50),  -- 'admin', 'manager', 'member'
    ...
);
```

### Role → Permission Mapping (Casbin)

```sql
-- casbin_rule table (auto-created by Casbin)
-- Stores role-permission policies
-- Example: ("p", "admin", "*", "employees", "create")
```

## Usage

### Assigning Roles (via API)

```bash
# Add user to a group with a role
POST /api/v1/users/{userId}/groups
{
    "group_id": 1,
    "role": "admin"
}
```

### Assigning Superadmin (via API)

```bash
# Set user as superadmin
PUT /api/v1/users/{userId}/superadmin
{
    "is_super_admin": true
}
```

### Programmatic Permission Check

```go
// In handlers/services, use PermissionService (injected via middleware context)
allowed, err := permissionService.CheckPermission(userID, orgID, "employees", "create")
```

## Initial Setup

### 1. Seed Role-Permission Policies

On first run, seed the Casbin policies:

```bash
SEED_RBAC_POLICIES=true ./kitamanager-api
```

This creates the role → permission mappings in `casbin_rule` table.

### 2. Create First Superadmin

After seeding policies, create a user and mark as superadmin:

```sql
UPDATE users SET is_super_admin = true WHERE id = 1;
```

Or via the API (requires existing superadmin):

```bash
PUT /api/v1/users/1/superadmin
{"is_super_admin": true}
```

## Casbin Configuration

### Model (`configs/rbac_model.conf`)

```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && (p.dom == r.dom || p.dom == "*") && r.obj == p.obj && r.act == p.act
```

- `sub`: Subject (role name, e.g., `admin`)
- `dom`: Domain (always `*` in our policies - org scoping is done via database)
- `obj`: Object (resource name)
- `act`: Action (create, read, update, delete)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RBAC_MODEL_PATH` | Path to Casbin model config | `configs/rbac_model.conf` |
| `SEED_RBAC_POLICIES` | Seed default policies on startup | `false` |

## Testing

Run RBAC tests:

```bash
# Unit tests for Casbin policy definitions
go test ./internal/rbac/... -v

# Middleware authorization tests
go test ./internal/middleware/... -v -run Authorization
```

The `Enforcer` includes testing methods (`AssignRole`, `CheckPermission`, etc.) that store role assignments directly in Casbin. These are used for unit testing policy definitions but are NOT used in production - production uses `PermissionService` with database-backed role assignments.
