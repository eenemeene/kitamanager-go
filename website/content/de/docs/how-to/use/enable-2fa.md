---
title: Zwei-Faktor-Authentifizierung aktivieren
weight: 12
---

Sie wollen einen zweiten Faktor (Authenticator-Code oder Sicherheitsschlüssel) zu Ihrem Konto hinzufügen, damit ein gestohlenes Passwort allein nicht ausreicht.

## TOTP aktivieren (Authenticator-App)

1. Öffnen Sie das Benutzermenü → **Einstellungen**.
2. In der Karte **Zwei-Faktor-Authentifizierung** klicken Sie auf **Zwei-Faktor-Authentifizierung aktivieren**.
3. Bestätigen Sie Ihr Passwort.
4. Scannen Sie den QR-Code mit Ihrer Authenticator-App (Google Authenticator, 1Password, Authy, Bitwarden etc.) — oder geben Sie den Schlüssel manuell ein.
5. Geben Sie den 6-stelligen Code aus der App ein und klicken Sie auf **Zwei-Faktor aktivieren**.
6. **Speichern Sie die danach gezeigten Wiederherstellungscodes**. Sie werden genau einmal angezeigt. Ausdrucken, im Passwort-Manager speichern oder aufschreiben. Ohne sie und ohne Authenticator muss ein Admin Ihr Konto zurücksetzen.
7. Häkchen setzen und auf **Fertig** klicken.

Die nächste Anmeldung fragt Passwort und danach einen 6-stelligen Code ab.

## Sicherheitsschlüssel hinzufügen (WebAuthn)

Nachdem TOTP an ist, hat dieselbe Karte **Sicherheitsschlüssel hinzufügen**:

1. Klicken Sie auf **Sicherheitsschlüssel hinzufügen**.
2. Der Browser öffnet seine WebAuthn-Aufforderung — tippen Sie auf Ihren YubiKey, stecken Sie Ihren Phone-Passkey ein, oder nutzen Sie das Plattform-Biometrie-System.
3. Optional einen Namen für den Schlüssel vergeben (z. B. „Yubikey-blau").

Sie können mehrere Schlüssel registrieren. Bei der nächsten Anmeldung können Sie wählen, welchen Faktor Sie verwenden.

## Weitere Aktionen

- **Wiederherstellungscodes neu erzeugen** — invalidiert die alten Codes und zeigt frische (auch nur einmalige Anzeige). Vorher Passwort bestätigen.
- **Zwei-Faktor-Authentifizierung deaktivieren** — entfernt jeden Faktor (TOTP und Sicherheitsschlüssel). Erfordert Passwort und einen aktuellen 2FA-Code.

## Hinweise

- 2FA ist *dringend empfohlen* für jedes Konto, das Daten bearbeiten kann — Admin, Manager, Personal. Mitglieder profitieren weniger von einem gestohlenen Konto, sollten es aber dennoch aktivieren.
- Authenticator und Wiederherstellungscodes verloren? Ein Admin kann Ihr **Passwort** zurücksetzen, aber heute nicht Ihre **MFA-Faktoren** zurücksetzen, ohne den Nutzer-Datensatz zu löschen. Verlieren Sie nicht beides.
- Für die team-weite Einführung (2FA als Standard für alle) siehe das Tutorial [Zwei-Faktor-Authentifizierung im Team einführen](../../../tutorials/enable-2fa-for-your-team/).
