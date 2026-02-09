---
title: Features
weight: 2
---

KitaManager Go provides a comprehensive set of features for managing kindergarten facilities.

## Organization Management

### Multi-Tenant Architecture
- Support multiple independent organizations
- Complete data isolation between organizations
- Organization-scoped access control
- State-level configuration (e.g., Berlin, Bavaria)

### Organizational Sections
- Group children and employees into departments
- Flexible section structure
- Capacity management

## Employee Management

### Complete Employee Database
- Personal information management
- Employment history tracking
- Flexible contract management
- Multiple concurrent contracts per employee

### Contract Management
- Define employment terms and conditions
- Position and grade tracking
- Weekly hours configuration
- Contract date validation (no overlaps)

### Pay Plan Integration
- Define pay scales and grades
- Step-based salary progression
- Grade-step combinations (e.g., S8a Step 3)

## Child Enrollment

### Comprehensive Child Records
- Personal and contact information
- Birthdate and gender tracking
- Enrollment history

### Enrollment Contracts
- Flexible contract periods
- Custom properties (care type, etc.)
- Automatic funding calculation
- Non-overlapping contract validation

### Funding Integration
- Automatic funding amount display
- Property-based funding calculations
- Government funding configuration

## Access Control

### Role-Based Permissions (RBAC)
Hybrid system combining Casbin policies with database-stored assignments.

| Role | Description |
|------|-------------|
| **Superadmin** | Full system access across all organizations |
| **Admin** | Full access within assigned organization(s) |
| **Manager** | Operational access (employees, children, contracts) |
| **Member** | Read-only access |

### Audit Logging
- Track all data changes
- User action history
- Compliance-ready audit trails

## Government Funding

### State Configuration
- Define funding rules per German state
- Property-based funding entries
- Flexible period management

### Funding Periods
- Time-based funding configurations
- Property combinations (e.g., care_type + ndh)
- Amount calculations in cents (precision)

### Automatic Calculations
- Real-time funding display
- Contract property matching
- Reporting capabilities

## API & Integration

### REST API
- Comprehensive REST API
- OpenAPI/Swagger documentation
- JWT authentication

### Security Features
- CORS configuration
- Rate limiting
- CSRF protection
- Request body size limits
- Security headers

## Modern Technology

### Performance
- Built with Go for high performance
- Efficient PostgreSQL queries
- Optimized frontend with Next.js

### Developer Experience
- Comprehensive documentation
- TypeScript frontend
- Hot reloading in development
- Extensive test coverage

### Deployment
- Docker support
- Single binary deployment
- Embedded frontend option
- Easy configuration
