# Architectural Review: KitaManager Go Backend API

*Reviewed: 2026-02-19*

## Executive Summary

This is a **well-engineered codebase** with strong fundamentals. The layered architecture (handlers → services → stores), consistent error handling, and comprehensive RBAC implementation demonstrate experienced engineering. However, there are several inconsistencies and design smells that should be addressed for a production system at scale.

**Overall Grade: B+** — Solid foundation, needs consistency cleanup.

---

## 1. CRITICAL: URL Parameter Naming Inconsistency

**Severity: Medium | Impact: API Consumer Confusion**

The API uses two contradictory conventions for URL parameters:

| Convention | Resources | Example |
|---|---|---|
| **Named** (`:userId`, `:groupId`, `:sectionId`) | Users, Groups, Sections | `/users/:userId` |
| **Generic** (`:id`) | Employees, Children, PayPlans, BudgetItems, Gov Fundings | `/employees/:id` |

The problem compounds with nested resources. When you have `/employees/:id/contracts/:contractId`, the parent uses generic `:id` but the child uses named `:contractId`. Compare with `/users/:userId/groups/:groupId` where both are named.

**Recommendation:** Standardize on named parameters throughout: `:employeeId`, `:childId`, `:payplanId`, `:budgetItemId`.

---

## 2. Statistics Endpoint Placement — Broken Information Architecture

**Severity: Medium | Impact: Inconsistent API Mental Model**

Statistics endpoints are split across two different locations with no clear rationale:

**Under `/organizations/:orgId/statistics/`** (StatisticsHandler):
```
GET /statistics/staffing-hours
GET /statistics/financials
GET /statistics/occupancy
GET /statistics/staffing-hours/employees
```

**Under `/organizations/:orgId/children/statistics/`** (ChildHandler):
```
GET /children/statistics/age-distribution
GET /children/statistics/contract-properties
```

**Under `/organizations/:orgId/children/`** (not even under statistics):
```
GET /children/funding
```

**Recommendation:** Consolidate all statistics under `/organizations/:orgId/statistics/`.

---

## 3. RBAC Permission Mismatch on Statistics

**Severity: Low | Impact: Confusing Authorization Model**

The org-level statistics endpoints require `children:read` permission for staffing hours and financials, but these are cross-domain statistics that also involve employee data.

---

## 4. Government Funding vs Government Funding Bills — Domain Confusion

**Severity: Medium | Impact: Confusing Domain Model**

| Resource | Scope | What It Is |
|---|---|---|
| `/government-fundings` | Global (superadmin) | Rate tables — defines how much the government pays per child type |
| `/organizations/:orgId/government-funding-bills` | Org-scoped | Uploaded ISBJ Excel billing documents |

The relationship between them is invisible in the API.

---

## 5. Government Funding Bills — RPC-style Route

**Severity: Low | Impact: Convention Violation**

`POST /government-funding-bills/isbj` uses an action verb in the URL. Should be `POST /government-funding-bills` with file format detection or as a form field.

---

## 6. Nested Resource CRUD Gaps — Government Funding Periods/Properties

**Severity: Low**

Government funding periods and properties have Create/Update/Delete but no Get or List endpoints, unlike PayPlan periods and BudgetItem entries which have both.

---

## 7. Handler Responsibility — ChildHandler Does Too Much

**Severity: Low | Impact: Maintainability**

The ChildHandler manages five distinct concerns: Child CRUD, contracts, age distribution statistics, contract properties distribution, and funding calculations.

---

## 8. Pagination Links Don't Include Filter Parameters

**Severity: Medium | Impact: Broken HATEOAS Navigation**

The pagination links only include `page` and `limit` parameters. Following a `_links.next` URL loses all active filters (search, section_id, active_on, etc.).

---

## 9. DTO Naming Deviation

**Severity: Low**

`UserOrganizationAddRequest` should be `UserAddOrganizationRequest` per the project's naming convention.

---

## 10. Store Interface Inconsistencies

**Severity: Low**

Some stores have dual methods for the same query (e.g., `FindByOrganization` and `FindByOrganizationPaginated`) while others only have the paginated version.

---

## 11. Contract Architecture — Implicit Polymorphism

**Severity: Informational**

`ContractProperties` is typed as `map[string]interface{}` with no schema validation. Works at current scale but may need schema validation as property types grow.

---

## What's Done Well

- **Consistent error handling**: Single `respondError()` function across all handlers
- **Pagination**: HATEOAS-style links, consistent `PaginatedResponse[T]` generic
- **Audit logging**: Every mutation consistently logged
- **HTTP status codes**: Uniformly correct (201/200/204)
- **Generic handler helpers**: `bindJSON[T]()`, `handleOrgList[T]()` reduce boilerplate
- **Health checks**: Proper K8s-ready liveness/readiness/health split
- **Security**: CSRF, rate limiting, security headers, token revocation
- **Compile-time interface checks**: `var _ Interface = (*Impl)(nil)` pattern

---

## Summary of Recommendations (Prioritized)

| # | Priority | Issue | Effort |
|---|---|---|---|
| 1 | **P1** | Fix pagination links to include filter params | Small |
| 2 | **P1** | Standardize URL parameter naming (all named or all generic) | Medium |
| 3 | **P2** | Consolidate statistics endpoints under `/statistics/` | Medium |
| 4 | **P2** | Rename `POST /government-funding-bills/isbj` to `POST /government-funding-bills` | Small |
| 5 | **P2** | Fix RBAC permissions on statistics | Small |
| 6 | **P3** | Rename `UserOrganizationAddRequest` → `UserAddOrganizationRequest` | Trivial |
| 7 | **P3** | Add missing GET endpoints for government funding periods/properties | Small |
| 8 | **P3** | Consider renaming `/government-fundings` to `/government-funding-rates` | Small (breaking) |
| 9 | **P3** | Extract `ChildStatisticsHandler` from `ChildHandler` | Small |
| 10 | **P4** | Consolidate store interface dual methods | Medium |
