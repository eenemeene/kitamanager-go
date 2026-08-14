---
title: API
weight: 1
aliases:
  - /docs/api/
---

KitaManager bietet eine REST-API. Die interaktive OpenAPI/Swagger-UI wird mit dem Server unter `/swagger/index.html` ausgeliefert — das ist die **maßgebliche** Referenz und entspricht exakt dem laufenden Build. Die Tabellen unten sind eine handgepflegte Zusammenfassung, die die Endpunkte nach Ressource gruppiert.

{{< callout type="info" >}}
Die vollständige englischsprachige Endpunkt-Übersicht steht unter [/en/docs/reference/api/](/en/docs/reference/api/). Für Authentifizierungsfluss, CSRF-Verhalten und Code-Beispiele bitte dort nachschlagen.
{{< /callout >}}

Die API ist **cookie-basiert** authentifiziert: ein erfolgreicher Login setzt ein HttpOnly `session`-Sitzungs-Cookie sowie ein per JS lesbares `csrf_token`-Cookie. Mutierende Anfragen (POST/PUT/PATCH/DELETE) müssen das CSRF-Token im `X-CSRF-Token`-Header zurückgeben. Es gibt keinen Refresh-Endpunkt — Sitzungen bleiben gültig bis zum Logout, Cookie-Ablauf, oder bis Sie sie unter `/me/sessions` widerrufen.

Vollständige Endpunkt-Tabellen mit Methoden und Pfaden: [englische API-Referenz](/en/docs/reference/api/).

## Fehlerantworten

Jeder Fehler ist ein Problem-Dokument nach
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) mit dem Content-Type
`application/problem+json`. Werten Sie `code` aus; zeigen Sie `detail` an oder
übersetzen Sie `code`, wenn Ihr Client nicht englischsprachig ist.

```json
{
  "type": "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#not_found",
  "title": "Resource not found",
  "status": 404,
  "detail": "child contract 7 not found",
  "instance": "/api/v1/organizations/1/children/42/contracts/7",
  "code": "not_found",
  "request_id": "0e03dc7d-9baa-4a23-a8ba-bc54ad5b30b9"
}
```

Alle Codes mit Bedeutung und Handlungsempfehlung: [Fehler](errors/).
