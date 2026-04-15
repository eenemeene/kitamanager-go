---
title: Architecture
weight: 2
---

KitaManager follows a clean architecture pattern with clear separation of concerns.

## System Overview

```mermaid
graph TB
    subgraph Frontend
        UI[Next.js 16 UI]
    end

    subgraph Backend
        API[Gin REST API]
        Auth[JWT Auth]
        RBAC[Casbin RBAC]
        Services[Business Logic]
        Store[Data Access Layer]
    end

    subgraph Database
        PG[(PostgreSQL)]
    end

    UI --> API
    API --> Auth
    API --> RBAC
    API --> Services
    Services --> Store
    Store --> PG
```

## RBAC Architecture

The application uses a hybrid RBAC system:

1. **Database** stores user-role-organization assignments (auditable, queryable)
2. **Casbin** stores role-permission mappings (optimized policy evaluation)

### Role Hierarchy

| Role | Scope | Permissions |
|------|-------|-------------|
| Superadmin | Global | Full system access |
| Admin | Organization | Full org access |
| Manager | Organization | Operational access |
| Member | Organization | Read-only access |
| Staff | Organization | Attendance management |

### Organization-Scoped Resources

Resources that belong to an organization use URL patterns:

```
/api/v1/organizations/{orgId}/employees
/api/v1/organizations/{orgId}/children
/api/v1/organizations/{orgId}/sections
```

## Report Tool

A standalone CLI tool (`tools/report-pdf/`) generates PDF reports by rendering the frontend's print pages via Playwright. It is **independent from the API and frontend** — it authenticates via HTTP and produces the same charts and tables users see in the browser.

```mermaid
graph LR
    Report[report-pdf Tool] -->|Login| API
    Report -->|Render print pages| UI
    Report -->|Send email| SMTP[SMTP Server]
```

The tool supports two modes:
- **One-shot**: Generate PDFs from the command line
- **Scheduled**: Run as a long-lived service, sending reports via email on a weekly or monthly schedule (configured via YAML file)

Reports are merged into a single PDF containing children, occupancy, staffing, and financials sections.

## Data Flow

1. **Request** arrives at Gin router
2. **Middleware** handles authentication and authorization
3. **Handler** validates input and calls service layer
4. **Service** implements business logic
5. **Store** performs database operations
6. **Response** is serialized and returned
