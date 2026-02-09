---
title: API Reference
weight: 3
---

KitaManager Go provides a comprehensive REST API with OpenAPI/Swagger documentation.

## API Documentation

The API documentation is available at `/swagger/index.html` when running the application.

## Authentication

All API endpoints (except login) require JWT authentication.

### Login

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "admin123"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Using the Token

Include the token in the `Authorization` header:

```bash
curl http://localhost:8080/api/v1/organizations \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

## API Endpoints

### Organizations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations` | List all organizations |
| POST | `/api/v1/organizations` | Create organization |
| GET | `/api/v1/organizations/{id}` | Get organization |
| PUT | `/api/v1/organizations/{id}` | Update organization |
| DELETE | `/api/v1/organizations/{id}` | Delete organization |

### Employees

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations/{orgId}/employees` | List employees |
| POST | `/api/v1/organizations/{orgId}/employees` | Create employee |
| GET | `/api/v1/organizations/{orgId}/employees/{id}` | Get employee |
| PUT | `/api/v1/organizations/{orgId}/employees/{id}` | Update employee |
| DELETE | `/api/v1/organizations/{orgId}/employees/{id}` | Delete employee |

### Employee Contracts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations/{orgId}/employees/{empId}/contracts` | List contracts |
| POST | `/api/v1/organizations/{orgId}/employees/{empId}/contracts` | Create contract |
| PUT | `/api/v1/organizations/{orgId}/employees/{empId}/contracts/{id}` | Update contract |
| DELETE | `/api/v1/organizations/{orgId}/employees/{empId}/contracts/{id}` | Delete contract |

### Children

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations/{orgId}/children` | List children |
| POST | `/api/v1/organizations/{orgId}/children` | Create child |
| GET | `/api/v1/organizations/{orgId}/children/{id}` | Get child |
| PUT | `/api/v1/organizations/{orgId}/children/{id}` | Update child |
| DELETE | `/api/v1/organizations/{orgId}/children/{id}` | Delete child |

### Child Contracts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations/{orgId}/children/{childId}/contracts` | List contracts |
| POST | `/api/v1/organizations/{orgId}/children/{childId}/contracts` | Create contract |
| PUT | `/api/v1/organizations/{orgId}/children/{childId}/contracts/{id}` | Update contract |
| DELETE | `/api/v1/organizations/{orgId}/children/{childId}/contracts/{id}` | Delete contract |

### Government Funding

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/government-fundings` | List funding configurations |
| POST | `/api/v1/government-fundings` | Create funding config |
| GET | `/api/v1/government-fundings/{id}` | Get funding config |
| DELETE | `/api/v1/government-fundings/{id}` | Delete funding config |

### Users & Groups

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List users |
| POST | `/api/v1/users` | Create user |
| GET | `/api/v1/groups` | List groups |
| POST | `/api/v1/groups` | Create group |

## Pagination

List endpoints support pagination with query parameters:

```bash
curl "http://localhost:8080/api/v1/organizations?page=1&limit=10"
```

Response includes pagination metadata:
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "limit": 10
}
```

## Error Responses

Errors are returned with appropriate HTTP status codes:

```json
{
  "error": "Description of the error"
}
```

| Status | Meaning |
|--------|---------|
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Missing or invalid token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 500 | Server Error - Internal error |
