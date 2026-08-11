---
title: 2FA im Team einführen
weight: 9
---

Sie wollen 2FA zum Standard für jedes Konto in Ihrer Organisation machen, das Daten bearbeiten kann.

## Schritte

1. **Zuerst auf Ihrem eigenen Konto aktivieren** nach [Zwei-Faktor-Authentifizierung aktivieren](../../use/enable-2fa/). Wiederherstellungscodes sicher speichern, durch Ab- und erneutes Anmelden verifizieren, dass die Code-Abfrage erscheint.
2. **Team-Standard festlegen** für die Authenticator-App (Google Authenticator, 1Password, Authy, Bitwarden — eine konsistente Wahl bedeutet eine einzige Anleitung) und für die Aufbewahrung der Wiederherstellungscodes (persönlicher Passwort-Manager bevorzugt).
3. **Jede:n Kolleg:in einrichten** persönlich oder per Bildschirmfreigabe. Gehen Sie [2FA aktivieren](../../use/enable-2fa/) durch, beobachten Sie das Speichern der Wiederherstellungscodes, melden Sie sie ab und wieder an, um die Code-Abfrage zu verifizieren.
4. **In das Onboarding aufnehmen**, sodass jede:r neue Admin/Manager:in 2FA am ersten Tag aktiviert, gleichzeitig mit dem Passwortwechsel.
5. **Quartalsweise prüfen:** es gibt noch keinen „Nutzer:innen ohne 2FA“-Bericht — schreiben Sie einen kurzen Check, der `GET /api/v1/users/{userId}/factors` für jede:n Admin/Manager:in (oder `me` für die aufrufende Person) abfragt und alle ohne aktivierten Faktor identifiziert.

## Hinweise

- Heute ist die einzige Wiederherstellung bei Verlust von Authenticator und Codes das Löschen und Neuanlegen des Kontos. Sagen Sie das beim Einrichten, damit Codes ernst genommen werden.
- Initiale Passwörter und Wiederherstellungscodes nicht per E-Mail kommunizieren — persönlich, per Videoanruf oder per Signal/Slack-DM.
- Für die verwandte Operator-Sorge (`TOTP_ENCRYPTION_KEY`-Rotation invalidiert jeden TOTP-Faktor) siehe [TOTP-Verschlüsselungsschlüssel rotieren](../../operate/rotate-totp-encryption-key/).
