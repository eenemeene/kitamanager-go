# KitaManager Go - Development Guidelines

## Dependency Versions

**Always use the latest versions** of all dependencies and tools. This includes Hugo, the Hextra theme, Go, Node.js, and all other libraries. Never pin to older versions or downgrade to work around compatibility issues — instead, upgrade the dependency chain to make everything work with the latest releases.

**Exception:** If a dependency's latest version breaks compatibility with another dependency that hasn't caught up yet (e.g., eslint-plugin-react not supporting eslint 10), pin to the latest compatible version and document the reason in the commit message. Track the upstream issue and upgrade once it's resolved.

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

The application uses Casbin for RBAC with organization-level multi-tenancy. See `website/content/en/docs/administration.md` (Role-Based Access Control section) for the canonical role and permission documentation.

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

### "Today" — always use `models.Today()`

Every "what calendar date is today?" decision MUST go through `models.Today()`. Never derive a date from `time.Now()` or `time.Now().UTC()` directly when you mean "today's date" — the answer must match the user's wall-clock day, not the server's clock day.

`models.Today()` returns the UTC midnight of the current calendar date in the application's timezone (`Europe/Berlin` by default, override via `KITAMANAGER_TIMEZONE`). The result is at UTC midnight so it composes with `models.TruncateToDate` and with DATE columns round-tripped through GORM.

```go
// CORRECT — Berlin's calendar today
today := models.Today()
if contract.From.Before(today) { ... }

// WRONG — server's UTC clock-day; off by one for ~1 hour every day
today := time.Now().UTC().Truncate(24 * time.Hour)
```

This rule applies to:
- Amend-mode threshold (is the contract started before *today*?)
- List defaults (`active_on=today` when the param is absent)
- Auto-derived dates (attendance recorded "today")
- Future-birthdate guards (is this birthdate after *today*?)
- Anywhere else "today" is implicit

`time.Now()` and `time.Now().UTC()` remain correct for instant timestamps (audit log times, JWT issued-at, MFA expiry, check-in/check-out times) — those want the precise moment, not a calendar day.

For tests that need to pin behavior to a specific instant or zone, call `models.DateIn(t, loc)` directly — it's the pure helper underneath `Today()`.

## E2E Testing

### Page Load Waits

**NEVER use `waitForLoadState('networkidle')`.** react-query keeps background requests active, which prevents pages from ever reaching networkidle state. This causes consistent 30-second timeouts in CI.

Use `waitForLoadState('load')` instead. For waiting on dynamic content, use element-level assertions with explicit timeouts:
```typescript
// CORRECT
await page.waitForLoadState('load');
await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible({ timeout: 10000 });

// WRONG - will timeout because react-query keeps fetching
await page.waitForLoadState('networkidle');
```

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

The primary use case is teachers and managers operating the app on **tablets** (768–1024px) and **phones** (375–414px) while standing in a group room. Touch comfort is a first-class requirement, not a polish pass.

### Breakpoint model (read this first)

| Prefix | Min Width | Covers                              | NOT                    |
|--------|-----------|-------------------------------------|------------------------|
| (none) | 0px       | Phones + iPad mini (744px portrait) | —                      |
| `md:`  | 768px     | **Tablet portrait** — iPad 10/11 (820), iPad Air 11 (820), iPad Pro 11 (834) | **NOT desktop** |
| `lg:`  | 1024px    | Tablet landscape, iPad Pro 13 / Air 13 portrait (1024), small laptops | — |
| `xl:`  | 1280px    | Desktop                             | —                      |

**The common mistake: treating `md:` as "desktop" and using it to shrink touch targets.** At 768–1023px the user is still tapping with a finger, not clicking with a mouse. If you want a compact desktop variant of something, gate it with `lg:` or `xl:`, never `md:`.

**The sidebar is the canonical example**: the shell forces the sidebar into icon-only rail mode at `md:` regardless of the user's stored collapse preference, and only renders the full labeled sidebar at `lg:+`. The `useIsLgUp()` hook in `@/hooks/use-media-query` exposes this as `isLgUp`, combined into an `effectiveCollapsed` value in `app-sidebar.tsx`. Any new sidebar-adjacent UI should follow the same pattern rather than keying off `sidebarCollapsed` directly.

### Touch targets — 44×44px minimum

The design-system primitives are already sized correctly:

| Primitive           | Height | Notes                                                        |
|---------------------|--------|--------------------------------------------------------------|
| `Button` (default)  | 44px   | `h-11`                                                       |
| `Button size="icon"`| 44×44  | `h-11 w-11` — use for all icon-only buttons                  |
| `Button size="lg"`  | 48px   | Primary CTAs / hero actions                                  |
| `Button size="sm"`  | 36px   | **Avoid for primary actions.** Only for dense desktop UI.    |
| `Button size="icon-sm"` | 36×36 | Only for truly dense cells (e.g., attendance week grid)   |
| `Input`             | 44px   | `h-11`                                                       |
| `SelectTrigger`     | 44px   | `h-11`                                                       |

Rules:
- Do **not** override these with `h-9`/`h-10` to match legacy mockups.
- Do **not** use `size="sm"` for "New X" / save / submit / primary-row actions — that's 36px and fails the minimum.
- Do **not** use a responsive shrink like `h-11 w-11 md:h-10 md:w-10` — see breakpoint model above.
- For a clickable bare `<button>` (inline editors, linkified text), set `min-h-9` plus `px-2 py-1` so the hit area clears 36px even when the text itself is small.
- For calendar day cells and similar dense grids, use `h-11 w-11 md:h-9 md:w-9` — full target on mobile, compact on desktop where mouse precision is available.

### Layout rules

- **Mobile-first stacking**: `flex-col` default, opt into `sm:flex-row` or `md:flex-row` for wider screens.
- **Responsive grids**: `grid-cols-1 md:grid-cols-2` — never fixed `grid-cols-2` / `grid-cols-3` for page-level layouts.
- **No fixed pixel widths on layout containers.** Icon buttons and stepper value labels (e.g., `min-w-[80px]`) are the exception.
- **Filter bars**: `flex flex-wrap gap-2` (or `gap-2 md:gap-4`) — controls must wrap on narrow screens.
- **Page headers**: stack title + actions on phone (`flex-col gap-3 sm:flex-row sm:items-center sm:justify-between`); truncate long titles (`truncate`) and let action groups `flex-wrap`.
- **Content padding**: `p-3 md:p-6` — don't waste horizontal space on phones.
- **Tables**: hide non-essential columns with `hidden md:table-cell`. The `<Table>` primitive already scrolls horizontally, so wide tables won't break the page, but forcing horizontal scroll for primary columns is a smell.

### Tables on narrow viewports

- Hide non-essential columns at md with `hidden md:table-cell` (sample: `children-table.tsx` hides Gender/Birthdate/Age on phone).
- **Action columns must wrap their buttons in a flex container.** Without it, a cell with 3+ icon buttons will wrap the icons vertically when the column runs out of width:
  ```tsx
  // CORRECT — buttons stay horizontal, gap is tight
  <TableCell className="text-right">
    <div className="flex flex-nowrap items-center justify-end gap-0.5">
      <Button size="icon">...</Button>
      <Button size="icon">...</Button>
    </div>
  </TableCell>

  // WRONG — buttons wrap to separate rows on tablet
  <TableCell className="text-right">
    <Button size="icon">...</Button>
    <Button size="icon">...</Button>
  </TableCell>
  ```
- When a row has many actions (>3) and the table shows at md, hide secondary ones with `className="hidden lg:inline-flex"` on the button. Keep only Edit + Delete inline at md.

### Verification

All new pages must be verified at **375px** (phone), **820px** (mainstream iPad portrait), and **1280px** (desktop). Use Playwright or the browser DevTools device toolbar. The E2E suite has `responsive.spec.ts` with `Responsive Layout - Mobile / Tablet / Desktop` describes as a template. The Tablet describe uses `hasTouch: true` and asserts touch-target sizes (≥44px) on the daily-use surfaces.

## Soft-Delete

As of migration 000015 the `users` and `organizations` tables are soft-deleted: `DELETE` at the app layer stamps `deleted_at` rather than physically removing the row, and subsequent queries that start from the GORM model auto-scope the tombstone out.

### The raw-query rule

**Any hand-written query (`.Table()`, `.Joins()`, `.Raw()`) that references the `users` or `organizations` table as a joined entity MUST explicitly filter out soft-deleted rows.** GORM's auto-scoping applies only to the primary model, not to joined tables.

Use the helpers in `internal/store/scoping.go`:

```go
// Good: raw JOIN through users, filter applied explicitly
q := db.Table("sessions").Joins("JOIN users ON users.id = sessions.user_id").Where(...)
err := store.ExcludeSoftDeletedUsers(q).Take(&row).Error

// Good: raw JOIN through organizations
err := store.ExcludeSoftDeletedOrganizations(q).Take(&row).Error

// Bad: forgotten filter — soft-deleted users would authenticate
db.Table("sessions").Joins("JOIN users ...").Where("sessions.id = ?", idHash).Take(...)
```

Queries that **start** from a GORM model (`db.First(&User{}, id)`, `db.Model(&User{}).Joins(...)`, etc.) auto-scope and need no helper.

### Admin / purge paths

Use `db.Unscoped()` explicitly for:
- Admin "trash view" endpoints that list tombstoned rows.
- `HardDelete` methods that physically remove rows for the Art. 17 erasure flow or the retention TTL cleanup.
- `FindByIDUnscoped` when a purge target might already be tombstoned.

Never call `.Unscoped()` in a default read path.

## Data Protection in Git

**NEVER commit real (non-anonymized) personal data to git or push it to GitHub.** This includes names, birthdates, addresses, voucher numbers, email addresses, financial amounts, or any other data from real Kitas, children, employees, or families. This applies to:

- Screenshots and images (always use seed data, never production data)
- Test fixtures and sample data
- Log output, error messages, or debug output pasted into commits or PRs
- Configuration files, database dumps, or export files
- Comments, commit messages, or PR descriptions

Violations of this rule expose the project to **DSGVO (GDPR) liability** — children's personal data receives the highest level of legal protection in Germany.

When you need example data, use the built-in seed data ("Kita Sonnenschein") which contains only fictional names and generated records.

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
