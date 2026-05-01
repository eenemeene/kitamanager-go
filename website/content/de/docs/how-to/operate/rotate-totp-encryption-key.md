---
title: TOTP-Verschlüsselungsschlüssel rotieren
weight: 5
---

Sie wollen `TOTP_ENCRYPTION_KEY` ändern (z. B. im Rahmen routinemäßiger Schlüsselrotation oder weil Sie ein Leak vermuten).

{{< callout type="warning" >}}
**Die Rotation invalidiert jeden gespeicherten TOTP-Schlüssel.** Jede Person mit eingerichtetem TOTP muss sich nach der Änderung neu einrichten. Planen Sie ein Roll-out-Fenster.
{{< /callout >}}

## Schritte

1. Neuen Schlüssel erzeugen:
   ```bash
   openssl rand -hex 32
   ```
2. Nutzer:innen informieren, dass 2FA zu einem festen Zeitpunkt neu eingerichtet werden muss.
3. `TOTP_ENCRYPTION_KEY` in Ihrer Umgebung aktualisieren (z. B. `.env`, Ihr Secrets-Manager).
4. API neu starten.
5. Bestehende TOTP-Faktoren sind nun nicht mehr entschlüsselbar. Nutzer:innen sehen ihre Karte noch, aber die Verifizierung schlägt fehl.
6. Begleiten Sie sie durch:
   - Anmelden (Passwort funktioniert wie zuvor).
   - Beim 2FA-Prompt einen **Wiederherstellungscode** eingeben — jede:r sollte einen haben.
   - Nach dem Anmelden in Einstellungen → 2FA → **Deaktivieren**, dann erneut **Aktivieren**, um neu einzurichten.

## Wenn Sie TOTP einer Person ohne deren Mitwirkung deaktivieren müssen

Heute gibt es keine Admin-„MFA zurücksetzen"-Oberfläche. Wenn eine Person nach der Rotation Authenticator und Wiederherstellungscodes verloren hat, ist der Wiederherstellungspfad:

1. Passwort der Person zurücksetzen (deaktiviert 2FA nicht).
2. Nutzer-Datensatz löschen.
3. Person mit derselben E-Mail und neuem Passwort neu anlegen.

Direkter Admin-Reset von MFA-Faktoren ist auf der Roadmap.

## Hinweise

- WebAuthn (Sicherheitsschlüssel) ist von der `TOTP_ENCRYPTION_KEY`-Rotation **nicht** betroffen. Nur TOTP-Geheimnisse werden mit diesem Schlüssel verschlüsselt.
- Für die Bedeutung des Schlüssels siehe [Architektur](../../../explanation/architecture/) und den Kommentarblock in `internal/config/config.go` rund um `TOTP_ENCRYPTION_KEY`.
- Schlüsselrotation ist auch ohne Verdacht eine sinnvolle Übung. Aufnehmen in Ihre jährliche Sicherheits-Checkliste.
