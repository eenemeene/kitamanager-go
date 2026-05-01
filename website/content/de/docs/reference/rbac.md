---
title: RBAC
weight: 4
---

KitaManager nutzt Casbin-basiertes, organisations-bezogenes RBAC. Es gibt fünf Rollen. Jede API-Anfrage löst die Rolle der aufrufenden Person für die angefragte Organisation auf und fragt dann Casbin, ob diese Rolle die Aktion auf der Ressource ausführen darf.

Für *wie man* Rollen vergibt, siehe [Nutzer:innen und Rollen verwalten](../../how-to/administer/). Für die Designentscheidung siehe [Architektur: RBAC](../../explanation/architecture/).

## Rollen

| Rolle | Geltungsbereich | Beschreibung |
|---|---|---|
| `superadmin` | Global | Vollzugriff auf das gesamte System über alle Organisationen. Kann Organisationen anlegen/löschen, Förder-Konfigurationen verwalten, das globale Audit-Log einsehen. |
| `admin` | Organisation | Vollzugriff innerhalb der zugewiesenen Organisation(en). Kann Mitarbeitende, Kinder, Verträge, Bereiche, Entgelttabellen und Nutzer:innen verwalten. Kann keine Organisationen anlegen/löschen oder Förder-Konfigurationen verwalten. |
| `manager` | Organisation | Operativer Zugriff innerhalb der zugewiesenen Organisation(en). Kann Mitarbeitende, Kinder und Verträge verwalten. Nur-Lesen auf Nutzer:innen, Bereichen, Entgelttabellen. |
| `member` | Organisation | Nur-Lese-Zugriff innerhalb der zugewiesenen Organisation(en). Kann Mitarbeitende, Kinder, Verträge, Bereiche, Entgelttabellen einsehen, aber nichts verändern. |
| `staff` | Organisation | Für Erzieher:innen, die Anwesenheit erfassen. Nur-Lesen auf Kindern, Betreuungsverträgen, Bereichen; volles CRUD nur auf Anwesenheit. |

## Berechtigungsmatrix

| Ressource | Superadmin | Admin | Manager | Member | Staff |
|----------|-----------|-------|---------|--------|-------|
| Organisationen | CRUD | Read/Update | Read | Read | Read |
| Mitarbeitende | CRUD | CRUD | CRUD | Read | — |
| Kinder | CRUD | CRUD | CRUD | Read | Read |
| Verträge | CRUD | CRUD | CRUD | Read | Read (nur Kind) |
| Anwesenheit | CRUD | CRUD | CRUD | Read | CRUD |
| Bereiche | CRUD | CRUD | Read | Read | Read |
| Förder-Konfigurationen | CRUD | — | — | — | — |
| Entgelttabellen | CRUD | CRUD | Read | Read | — |
| Haushaltsposten | CRUD | CRUD | Read | Read | — |
| Statistiken | Read | Read | Read | Read | — |
| Nutzer:innen | CRUD | CRUD | Read | — | — |
| ISBJ-Förderabrechnungen | Create / Read / Delete | Create / Read / Delete | Create / Read / Delete | — | — |
| Audit-Log (org-bezogen) | Read | Read | — | — | — |

**Geltungsbereich:** `superadmin` operiert über alle Organisationen. Alle anderen Rollen sind auf die Organisationen beschränkt, in denen sie Mitglied sind. Eine Person kann Mitglied mehrerer Organisationen mit unterschiedlicher Rolle pro Organisation sein (z. B. Admin in Kita A, Manager in Kita B).

## URL-Schema für organisations-bezogene Ressourcen

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

Globale Ressourcen (kein `orgId` in der URL): `/api/v1/organizations`, `/api/v1/users`, `/api/v1/government-funding-rates`, `/api/v1/audit-logs` (Superadmin), `/api/v1/me/...`.

## Authorization-Middleware

Handler erklären ihre Anforderungen mit:

```go
authzMiddleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead)
authzMiddleware.RequireSuperAdmin()
```

Die Middleware extrahiert die Sitzung des Nutzers, löst dessen Rolle für die `orgId` der URL auf (falls vorhanden) und fragt Casbin. Ein Fehlschlag ergibt `403 Forbidden`.
