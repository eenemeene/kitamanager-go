---
title: Fehler
weight: 3
---

Jeder API-Fehler ist ein Problem-Dokument nach
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) und wird mit
`Content-Type: application/problem+json` ausgeliefert. Das gilt auch für die
Antworten, die der Router selbst erzeugt — unbekannter Pfad, falsche Methode,
Panic —, sodass ein Client für jeden Fehlerfall dieselbe Struktur auswerten kann
und nie einen Klartext-Body behandeln muss.

{{< callout type="info" >}}
Die API ist einsprachig englisch: `title` und `detail` sind englische Sätze. Für
Endnutzerinnen und Endnutzer übersetzt die Oberfläche den `code` — deshalb sehen
Sie in KitaManager deutsche Fehlermeldungen, obwohl die API englisch antwortet.
Die Feldnamen und Codes unten bleiben aus demselben Grund englisch.
{{< /callout >}}

```json
{
  "type": "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#contract_overlap",
  "title": "Contract periods overlap",
  "status": 409,
  "detail": "contract periods overlap between 2026-01-01 and 2026-03-31",
  "instance": "/api/v1/organizations/1/children/42/contracts/7/amend",
  "code": "contract_overlap",
  "request_id": "0e03dc7d-9baa-4a23-a8ba-bc54ad5b30b9"
}
```

## Welches Feld wofür

| Feld | Verwendung |
|---|---|
| `code` | **Fallunterscheidung im Code.** Stabile Kennung; ändert sich nicht, wenn eine Meldung umformuliert wird. |
| `type` | Link auf diese Seite, verankert am Code. Dieselbe Information wie `code`, in der von der Spezifikation vorgesehenen Form. |
| `title` | Kurze Beschreibung des Fehlertyps. Für einen Code immer identisch. |
| `detail` | Was in *diesem* Fall passiert ist — welcher Vertrag, welche Daten, welches Feld. |
| `instance` | Der angefragte Pfad. |
| `request_id` | Bei einer Fehlermeldung bitte mit angeben; sie entspricht der Zeile im Server-Log. |
| `invalid_params` | Bei Validierungsfehlern: ein Eintrag je abgelehntem Feld. |

## Validierungsfehler

Ein 400 mit `code: validation_error` benennt die abgelehnten Felder, sodass ein
Formular die betroffenen Eingaben markieren kann, statt einen Satz über allen
Feldern auszugeben.

```json
{
  "type": "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#validation_error",
  "title": "Validation failed",
  "status": 400,
  "detail": "email must be a valid email address; weekly_hours is required",
  "code": "validation_error",
  "invalid_params": [
    { "field": "email", "reason": "must be a valid email address" },
    { "field": "weekly_hours", "reason": "is required" }
  ]
}
```

`field` ist der JSON-Pfad im Request-Body; verschachtelte Felder erscheinen also
als `properties.0.name`.

## Codes

### not_found

**404.** Die adressierte Ressource existiert nicht — oder sie liegt außerhalb der
Organisationen, die diese Sitzung sehen darf. Beides ist bewusst nicht
unterscheidbar, sodass das Durchprobieren von IDs keine Information preisgibt.

### bad_request

**400.** Die Anfrage war nicht auswertbar: fehlerhaftes JSON, ein unbekanntes
Feld (Bodies weisen nicht deklarierte Felder zurück), mehr als ein JSON-Wert oder
ein Body über der Größengrenze.

### validation_error

**400.** Der Body war lesbar, hat aber die Feldregeln verletzt. Siehe
`invalid_params` oben.

### conflict

**409.** Die Anfrage widerspricht dem aktuellen Zustand der Ressource.

### contract_overlap

**409.** Die Vertragszeiträume würden denselben Tag doppelt abdecken. Passen Sie
die Daten so an, dass die Zeiträume aneinander anschließen statt sich zu
überschneiden — oder verwenden Sie den Boundary-Endpunkt, der die Grenze zwischen
zwei Zeiträumen in einem Aufruf verschiebt und konstruktionsbedingt keine
Überschneidung erzeugen kann.

### email_conflict

**409.** Diese E-Mail-Adresse ist bereits einem anderen Benutzer zugeordnet.

### duplicate_bill_hash

**409.** Diese Rechnungsdatei wurde bereits importiert. Geprüft wird der Hash des
Dateiinhalts — eine umbenannte Kopie wird also ebenfalls erkannt.

### duplicate_bill_month

**409.** Für diesen Monat existiert bereits eine Rechnung.

### unauthorized

**401.** Keine Sitzung, abgelaufene Sitzung oder nicht verifizierte
Zugangsdaten. Clients sollten den lokalen Sitzungszustand verwerfen und zur
Anmeldung leiten — mit einer Ausnahme: Ein 401 von `/login`, `/logout` oder den
MFA-Endpunkten gehört zum normalen Ablauf dieser Endpunkte und bedeutet nicht,
dass eine bestehende Sitzung ungültig geworden ist.

### forbidden

**403.** Authentifiziert, aber nicht berechtigt: Der Rolle fehlt die Berechtigung,
oder die Anfrage greift über die Organisationen der Sitzung hinaus. Ebenfalls von
der CSRF-Prüfung verwendet, wenn eine per Cookie authentifizierte schreibende
Anfrage ohne passenden `X-CSRF-Token` eintrifft.

### precondition_required

**428.** Ein Schreibzugriff mit optimistischer Sperrung kam ohne `If-Match`-Header
an. Lesen Sie die Ressource, übernehmen Sie ihren `ETag` (oder das Feld `version`
im Body) und senden Sie ihn mit.

### precondition_failed

**412.** Die Version im `If-Match` ist nicht mehr aktuell: Jemand anderes hat die
Ressource seit dem Lesen geändert. Neu laden, die Änderung anzeigen, dann erneut
anwenden. Das ist bewusst von `contract_overlap` getrennt — hier ein veralteter
Lesestand, dort ein ungültiger Zeitraum —, damit ein Client „neu laden“ von
„Daten korrigieren“ unterscheiden kann.

### too_many_requests

**429.** Ratenbegrenzung. Der Header `Retry-After` enthält die Wartezeit in
Sekunden.

### method_not_allowed

**405.** Der Pfad existiert, die Methode nicht.

### internal_error

**500.** Serverseitiger Fehler. `detail` ist bewusst ein fester Text — die
zugrunde liegende Fehlermeldung kann ein Query-Fragment oder eine
Datenbankmeldung enthalten, und die gehört ins Log, nicht in eine Antwort. Über
die `request_id` lassen sich beide zusammenführen; bitte geben Sie sie in einer
Fehlermeldung an.
