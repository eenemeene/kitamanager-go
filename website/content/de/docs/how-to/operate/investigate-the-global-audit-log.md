---
title: Globales Audit-Log untersuchen
weight: 6
---

Sie sind Superadmin und wollen organisations-übergreifend ermitteln, oder Login-/Auth-Ereignisse einsehen, die das org-bezogene Audit-Log ausblendet.

Das globale Audit-Log ist heute **nur per API** zugänglich. Es gibt keine dedizierte UI-Seite; Superadmins fragen die API direkt an.

## Schritte

```bash
# Alle Ereignisse zwischen zwei Daten
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs?from=2026-04-01&to=2026-04-30"

# Nach Aktion-Teilstring filtern (z. B. alle Logins)
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs?action=login"

# Nach Akteur-Nutzer-ID filtern
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs?user_id=42"

# Einen bestimmten Eintrag holen
curl -b cookies.txt "http://localhost:8080/api/v1/audit-logs/12345"
```

Für Request-/Response-Details siehe [API: Audit-Logs](../../../reference/api/).

## Was im globalen Log steht, das im org-bezogenen Log nicht steht

- `login_success`, `login_failed` (mit der IP, die es versucht hat)
- `password_change_self`, `password_change_admin`
- `mfa_factor_create`, `mfa_factor_delete`, `mfa_verify_failed`
- Organisations-übergreifende Aktionen (Org anlegen/löschen, Superadmin-Vergabe/Widerruf, Förder-Sätze-Updates)

## Hinweise

- Der org-bezogene Log unter `/api/v1/organizations/{orgId}/audit-logs` blendet Login-/Passwort-Ereignisse *absichtlich* aus, weil sie organisations-übergreifend sensibel sind. Eine Person, die Admin in zwei Kitas ist, soll nicht aus dem Log einer Org erfahren, wann sie sich für die andere angemeldet hat.
- Audit-Einträge sind append-only. Es gibt keine API zum Ändern oder Löschen.
- Für Routine-Audit-Anfragen (wer hat in unserer Org letzte Woche was gelöscht) reicht der org-bezogene Log über die UI — siehe [Audit-Log der Organisation prüfen](../../administer/review-audit-log/).
