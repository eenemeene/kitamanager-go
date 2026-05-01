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

Die API ist **cookie-basiert** authentifiziert: ein erfolgreicher Login setzt ein HttpOnly `access_token`-Sitzungs-Cookie sowie ein per JS lesbares `csrf_token`-Cookie. Mutierende Anfragen (POST/PUT/PATCH/DELETE) müssen das CSRF-Token im `X-CSRF-Token`-Header zurückgeben. Es gibt keinen Refresh-Endpunkt — Sitzungen bleiben gültig bis zum Logout, Cookie-Ablauf, oder bis Sie sie unter `/me/sessions` widerrufen.

Vollständige Endpunkt-Tabellen mit Methoden und Pfaden: [englische API-Referenz](/en/docs/reference/api/).
