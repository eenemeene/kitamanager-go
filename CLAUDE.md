# KitaManager Go - Development Guidelines

## Dependency Versions

**Always use the latest versions** of all dependencies and tools. This includes Hugo, the Hextra theme, Go, Node.js, and all other libraries. Never pin to older versions or downgrade to work around compatibility issues — instead, upgrade the dependency chain to make everything work with the latest releases.

## Go Best Practices

**Always use the latest Go language features and idioms.** When writing or modifying Go code, prefer modern standard library packages and syntax over older patterns.

### Required Modern Patterns

- **`any` over `interface{}`**: Always use `any` (available since Go 1.18). Never use `interface{}`.
- **`slices` package over `sort.Slice`**: Use `slices.SortFunc` with `cmp.Compare` and `cmp.Or` for sorting. Use `slices.Contains`, `slices.Index`, etc. instead of manual loops.
- **`maps` package**: Use `maps.Keys`, `maps.Values`, `maps.Clone` etc. instead of manual map iteration. Combine with `slices.Collect` to collect iterators into slices.
- **`cmp` package**: Use `cmp.Compare` for comparisons in sort functions. Use `cmp.Or` for multi-field sorting.
- **`for range N`**: Use `for range N` (Go 1.22+) instead of `for i := 0; i < N; i++` for counting loops. Use `for i := range N` when the index is needed.
- **`strings.Repeat`**: Use `strings.Repeat(s, n)` instead of loop-based string building for simple repetitions.
- **`log/slog`**: Use structured logging with `slog` (Go 1.21+). Never use `log.Printf` or `fmt.Printf` for application logging.

### Examples

```go
// CORRECT - modern Go
slices.SortFunc(items, func(a, b Item) int {
    return cmp.Or(
        cmp.Compare(a.Name, b.Name),
        cmp.Compare(a.ID, b.ID),
    )
})
values := slices.Collect(maps.Values(myMap))
if slices.Contains(roles, targetRole) { ... }
for i := range 10 { ... }

// WRONG - outdated patterns
sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
var values []V; for _, v := range myMap { values = append(values, v) }
for _, r := range roles { if r == targetRole { ... } }
for i := 0; i < 10; i++ { ... }
```

## API Handler Documentation

All API handlers MUST be documented using swaggo annotations. This enables automatic OpenAPI/Swagger specification generation.

### Required Annotations

Every handler function must include the following annotations:

```go
// HandlerName godoc
// @Summary Short description of the endpoint
// @Description Detailed description of what the endpoint does
// @Tags tag-name
// @Accept json
// @Produce json
// @Security BearerAuth  // For protected endpoints
// @Param paramName path/query/body type required "Description"
// @Success statusCode {object/array} ResponseType
// @Failure statusCode {object} ErrorResponse
// @Router /api/v1/path [method]
func (h *Handler) HandlerName(c *gin.Context) {
    // implementation
}
```

### Example

```go
// Create godoc
// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserCreateRequest true "User data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
    // implementation
}
```

### Request/Response Types

All request and response structs should include `example` tags for better documentation.

#### DTO Naming Convention

All DTOs (Data Transfer Objects) must follow a consistent naming pattern.

**Request DTOs** - Use `{Resource}{Action}Request`:
- Create: `UserCreateRequest`, `ChildCreateRequest`, `EmployeeContractCreateRequest`
- Update: `UserUpdateRequest`, `ChildUpdateRequest`, `FundingPeriodUpdateRequest`
- Other actions: `AssignFundingRequest`, `SetSuperAdminRequest`

**Response DTOs** - Use `{Resource}Response`:
- `UserResponse`, `ChildResponse`, `EmployeeContractResponse`, `FundingPeriodResponse`

**Nested resources follow the same pattern**:
- `ChildContractCreateRequest` (not `CreateChildContractRequest`)
- `FundingEntryUpdateRequest` (not `UpdateFundingEntryRequest`)

**DO NOT** use these incorrect patterns:
- `Create{Resource}Request` (wrong: `CreateUserRequest`)
- `Update{Resource}Request` (wrong: `UpdateUserRequest`)
- `{Resource}Create` (wrong: `UserCreate` - missing `Request` suffix)

```go
type UserCreateRequest struct {
    Name     string `json:"name" binding:"required" example:"John Doe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Password string `json:"password" binding:"required,min=6" example:"secret123"`
}

type UserResponse struct {
    ID    uint   `json:"id" example:"1"`
    Name  string `json:"name" example:"John Doe"`
    Email string `json:"email" example:"john@example.com"`
}
```

### Generating Documentation

Run the following command to generate/update the OpenAPI specification:

```bash
swag init -g cmd/api/main.go
```

This will create/update files in the `docs/` directory.

## RBAC (Role-Based Access Control)

The application uses Casbin for RBAC with organization-level multi-tenancy. See `docs/RBAC.md` for full documentation.

### Roles

- `superadmin` - Full system access across all organizations
- `admin` - Full access within assigned organization(s)
- `manager` - Operational access (employees, children, contracts)

### Organization-Scoped Routes

Resources that belong to an organization use the URL pattern:
```
/api/v1/organizations/{orgId}/employees
/api/v1/organizations/{orgId}/children
```

### Authorization Middleware

Use the authorization middleware to protect routes:

```go
// Require specific permission
authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead)

// Require superadmin
authzMiddleware.RequireSuperAdmin()
```

## Container Images

`Dockerfile.api` and `Dockerfile.frontend` are the **single source of truth** for all OCI/Docker images. Both use [Chainguard](https://www.chainguard.dev/) base images for minimal, secure containers.

- `Dockerfile.api` — Multi-stage build: `cgr.dev/chainguard/go` (builder) + `cgr.dev/chainguard/static` (runtime)
- `Dockerfile.frontend` — Multi-stage build: `cgr.dev/chainguard/node` (builder + runtime)

Container images are the **only release artifacts**. The release workflow builds and pushes multi-arch images to GHCR when a GitHub release is published. No standalone binary artifacts are produced.

## Database Schema Changes

When making changes to the database schema (models), you MUST:

1. **Handle migrations** - Create proper database migrations for any schema changes. Never rely solely on GORM AutoMigrate for production changes.

2. **Update the schema diagram** - Regenerate the database diagram in `docs/` using:
   ```bash
   tbls doc --force postgres://user:pass@localhost:5432/kitamanager docs/schema
   ```
   Or configure `.tbls.yml` for consistent settings.

### Schema Diagram Tool

The project uses [tbls](https://github.com/k1LoW/tbls) to auto-generate database documentation including ER diagrams.

Install: `go install github.com/k1LoW/tbls@latest`

## Currency Storage

**All monetary values MUST be stored as integers in cents** (or the smallest currency unit).

This avoids floating-point precision errors that occur with decimal arithmetic:
```go
// Floating point - WRONG
0.1 + 0.2 = 0.30000000000000004

// Cents as integers - CORRECT
10 + 20 = 30
```

### Examples

| EUR Amount | Stored Value (cents) |
|------------|---------------------|
| €1,668.47  | 166847              |
| €0.01      | 1                   |
| €100.00    | 10000               |

### In Code

```go
// Model - store as int
type FundingProperty struct {
    Payment int `gorm:"not null" json:"payment"` // cents
}

// Converting EUR to cents
func euroToCents(eur float64) int {
    return int(math.Round(eur * 100))
}

// Display in frontend (TypeScript)
function formatCurrency(cents: number): string {
    return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })
}
```

### When Importing Data

When importing monetary data from external sources (YAML, CSV, APIs), always convert to cents before storage:
```go
payment := int(math.Round(yamlProperty.Payment * 100))
```

## Date/Time Handling

**Always use proper date/time objects.** Never use strings or regex to parse, compare, or manipulate dates and times.

### Go

Use `time.Time` and the `time` package for all date/time operations:

```go
// CORRECT - use time.Time
from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
if contract.From.Before(child.Birthdate) { ... }

// WRONG - string comparison or regex
if contractFrom < "2024-01-01" { ... }
re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
```

### TypeScript

Use `Date` objects or a date library (e.g., `date-fns`):

```typescript
// CORRECT - Date objects
const from = new Date('2024-01-01');
if (from < birthdate) { ... }

// WRONG - string manipulation
if (dateStr.split('-')[0] === '2024') { ... }
```

## E2E Testing

### Language/Locale

**ALWAYS use English locale for E2E tests.** This ensures consistent text matching regardless of the developer's system locale.

```typescript
// At the top of each test file
test.use({ locale: 'en-US' });
```

Use English text in test assertions and test data (e.g., "Deputy Manager" not "Gruppenleitung").

### Avoid Date-Dependent Assertions

**Do NOT test status values that depend on "today's date"** (e.g., "Active", "Upcoming", "Ended"). These tests become flaky over time as dates pass.

Instead:
- Use fixed past dates (e.g., `2024-01-01`) when creating test data
- Test that the data appears correctly, not its computed status
- If status must be tested, mock the date or use date ranges that won't expire

```typescript
// BAD - will fail when 2024-06-01 passes
await page.getByLabel(/Start Date/i).fill('2024-06-01');
await expect(page.getByText('Upcoming')).toBeVisible();

// GOOD - test the data, not the status
await page.getByLabel(/Start Date/i).fill('2024-01-01');
await expect(page.getByText(/fulltime/i)).toBeVisible();
```

## REST API Conventions

### Resource-Oriented Endpoints

Use resource-oriented URLs. **Do NOT use RPC-style action verbs** in endpoint paths.

```
# GOOD - resource-oriented
POST   /children/:id/attendance          (create)
PUT    /children/:id/attendance/:attendanceId  (update)

# BAD - RPC-style action verbs
POST   /children/:id/attendance/check-in
PUT    /children/:id/attendance/:id/check-out
POST   /children/:id/attendance/absent
```

### URL Parameter Naming

For nested resources, use `:id` for the parent resource and a named param (`:contractId`, `:attendanceId`, `:periodId`) for the sub-resource. This matches how Gin resolves route parameters.

```
/organizations/:orgId/employees/:id/contracts/:contractId
/organizations/:orgId/children/:id/attendance/:attendanceId
/organizations/:orgId/children/:id/contracts/:contractId
```

### HTTP 204 No Content

When returning `204 No Content`, do NOT include a response body. Use `c.Status()` instead of `c.JSON()`:

```go
// CORRECT
c.Status(http.StatusNoContent)

// WRONG - sends a body with 204
c.JSON(http.StatusNoContent, nil)
```

### Required Query Parameters

Required query parameters MUST be validated and return an error when missing. Do NOT silently default them.

```go
// CORRECT - validates required params
from, ok := parseRequiredDate(c, "from")

// WRONG - silently defaults to today when param is required
from, ok := parseOptionalDate(c, "from")
```

### Audit Logging

All mutating handlers MUST include audit logging, at minimum for delete operations. Follow the existing pattern:

```go
// Get resource info before deletion for audit log
resource, err := h.service.GetByID(ctx, id, orgID, parentID)
// ... perform delete ...
// Log the deletion
h.auditService.LogResourceDelete(actorID, "resource_type", id, resourceName, c.ClientIP())
```

## Responsive Design

All frontend components MUST be mobile-friendly. The app is used by teachers on tablets and phones.

### Required Practices

- **Mobile-first layouts**: Use `flex-col` by default, add `md:flex-row` for wider screens
- **Responsive grids**: Use `grid-cols-1 md:grid-cols-2` instead of fixed `grid-cols-2`
- **No fixed pixel widths** for layout containers (exception: icon buttons)
- **Touch targets**: Minimum 44x44px (`h-11 w-11`) for interactive elements on mobile
- **Content padding**: Use `p-3 md:p-6` to preserve space on small screens
- **Table columns**: Hide non-essential columns on mobile with `hidden md:table-cell`
- **Filter bars**: Use `flex flex-wrap gap-2` so controls wrap on narrow screens
- **Test viewports**: All new pages must work at 375px (phone), 768px (tablet), and 1280px (desktop)

### Breakpoint Reference

| Prefix | Min Width | Use Case |
|--------|-----------|----------|
| (none) | 0px       | Mobile phones (default) |
| `md:`  | 768px     | Tablets, small laptops |
| `lg:`  | 1024px    | Desktop |

## Git Workflow

**Always work on a feature branch** and submit changes via pull request. Never commit directly to `main`.

```bash
# CORRECT - feature branch + PR
git checkout -b feat/my-feature
# ... make changes ...
git push -u origin feat/my-feature
gh pr create --fill

# WRONG - direct commit to main
git commit -m "..." && git push
```

Merge the PR only after all GitHub CI checks pass.

## Releases

**Always create a GitHub release using `gh release create`**, never a bare git tag. Creating a GitHub release automatically creates the underlying git tag and triggers the release workflow to build and push container images.

```bash
# CORRECT - creates release + tag, triggers container image builds
gh release create v1.2.3 --generate-notes

# WRONG - bare tag, no GitHub release, container images won't be built
git tag v1.2.3
git push origin v1.2.3
```
