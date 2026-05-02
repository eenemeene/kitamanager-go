---
title: Audit-Log-Aktions-Codes
weight: 6
---

Jeder Audit-Log-Eintrag trägt ein `action`-Feld. Das sind die möglichen Werte. Verwenden Sie sie zum Filtern des Audit-Logs über das **Aktion**-Feld der UI oder den `?action=…`-Parameter der API (Substring-Match).

## Authentifizierung und Sitzungen

| Code | Wann emittiert |
|---|---|
| `login` | Erfolgreiche Anmeldung (nach MFA falls aktiviert). |
| `login_failed` | Falsches Passwort oder unbekannte E-Mail. |
| `login_mfa_required` | Erster Schritt erfolgreich; MFA wird erwartet. |
| `logout` | Nutzer:in hat sich abgemeldet. |
| `session_revoked` | Eine Sitzung wurde beendet (vom Nutzer oder durch eine andere Anmeldung). |

## Multi-Faktor-Authentifizierung

| Code | Wann emittiert |
|---|---|
| `factor_enrolled` | TOTP- oder WebAuthn-Faktor aktiviert. |
| `factor_deleted` | Nutzer:in hat einen eigenen Faktor entfernt. |
| `factor_admin_deleted` | Admin hat den Faktor einer Nutzer:in entfernt. |
| `factor_label_updated` | Faktor umbenannt (z. B. „Yubikey-blau"). |
| `factor_activation_locked` | Zu viele falsche Codes beim Einrichten. |
| `backup_codes_regenerated` | Nutzer:in hat Wiederherstellungscodes neu erzeugt. |
| `mfa_challenge_succeeded` | TOTP-/Backup-Code-/WebAuthn-Schritt bestanden. |
| `mfa_challenge_failed` | MFA-Verifizierung fehlgeschlagen. |
| `mfa_challenge_locked` | Zu viele falsche MFA-Versuche; Konto temporär gesperrt. |

## Passwörter

| Code | Wann emittiert |
|---|---|
| `password_change` | Self-Service-Passwortwechsel erfolgreich. |
| `password_change_failed` | Self-Service-Wechsel abgelehnt (falsches aktuelles Passwort etc.). |
| `password_reset` | Admin hat Passwort einer anderen Nutzer:in zurückgesetzt. |
| `password_reset_failed` | Admin-Reset am `actor_password`-Step-up gescheitert. `user_id` ist die handelnde Person, `resource_id` das Ziel. Steuert die per-Akteur-Sperre. |

## Nutzer:innen und Organisationen

| Code | Wann emittiert |
|---|---|
| `user_create` | Neues Nutzer-Konto angelegt. |
| `user_delete` | Nutzer:in soft-gelöscht. |
| `user_purged` | Nutzer:in hart gelöscht (DSGVO Art. 17 oder TTL-Purge). |
| `user_add_to_org` | Nutzer:in eine Rolle in einer Organisation zugewiesen. |
| `user_remove_from_org` | Mitgliedschaft entfernt. |
| `role_change` | Rolle innerhalb einer Organisation geändert. |
| `superadmin_grant` | Superadmin-Status vergeben. |
| `superadmin_revoke` | Superadmin-Status entzogen. |
| `superadmin_change_failed` | Superadmin-Vergabe/-Entzug am `actor_password`-Step-up gescheitert. `user_id` ist die handelnde Person, `resource_id` das Ziel. |
| `org_create` | Organisation angelegt (nur Superadmin). |
| `org_delete` | Organisation soft-gelöscht. |
| `org_purged` | Organisation hart gelöscht. |

## System

| Code | Wann emittiert |
|---|---|
| `audit_log_purged` | Retention-TTL hat alte Audit-Zeilen entfernt. `details` enthält `deleted_rows` und das `older_than`-Cutoff. |

## Ressourcen

Mutierende Operationen auf den meisten Ressourcen emittieren eine Aktion der Form `<ressource>_<verb>` — `child_create`, `child_update`, `child_delete`, `employee_create`, `government_funding_bill_create` etc. Per Substring auf den Ressourcen-Namen filtern (`child`, `employee`, `government_funding_bill`), um sie alle zu sehen.

`employee_delete` und `child_delete` sind oben separat aufgeführt, weil sie bei Triagen am häufigsten geprüft werden.

## Hinweise

- Der org-bezogene Audit-Log unter `Einstellungen → Protokoll` blendet absichtlich Login-/Passwort-/MFA-Ereignisse aus, weil sie organisations-übergreifend sensibel sind. Superadmins sehen sie über `GET /api/v1/audit-logs` — siehe [Globales Audit-Log untersuchen](../../how-to/operate/investigate-the-global-audit-log/).
- Die obige Liste ist der Stand zum Zeitpunkt des Schreibens; die maßgebliche Quelle ist `internal/models/audit.go`.
