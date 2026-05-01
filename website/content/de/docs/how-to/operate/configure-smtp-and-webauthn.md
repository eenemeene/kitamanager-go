---
title: SMTP und WebAuthn konfigurieren
weight: 10
---

Sie wollen E-Mail-getriebene Funktionen (derzeit optional) und Sicherheitsschlüssel-Anmeldung aktivieren.

## SMTP (optional)

Wenn `SMTP_HOST` leer ist, sind E-Mail-bezogene Funktionen aus. Zum Aktivieren setzen:

```
SMTP_HOST=smtp.example.org
SMTP_PORT=587
SMTP_USER=kitamanager@example.org
SMTP_PASSWORD=<App-Passwort>
SMTP_FROM=KitaManager <noreply@example.org>
```

API neu starten. Die erste E-Mail-auslösende Aktion zeigt SMTP-Fehler im API-Log, falls eines davon falsch ist; es gibt keinen Verbindungs-Check beim Start.

## WebAuthn (Sicherheitsschlüssel)

Drei Env-Vars müssen zur öffentlichen URL passen, die der Browser sieht:

```
WEBAUTHN_RP_ID=kitamanager.example.org              # nur Host, kein Schema, kein Slash
WEBAUTHN_RP_NAME=KitaManager — Kita Sonnenschein    # in der Browser-Aufforderung gezeigt
WEBAUTHN_ORIGINS=https://kitamanager.example.org    # voller Origin, komma-getrennt für mehrere
```

Nach Neustart funktioniert die **Sicherheitsschlüssel hinzufügen**-Schaltfläche auf der Einstellungsseite. Wenn sie still scheitert oder der Browser sagt „der Sicherheitsschlüssel ist für diese Seite nicht gültig", passt die RP ID nicht zur URL.

## Hinweise

- Eine Änderung von `WEBAUTHN_RP_ID` invalidiert jeden zuvor registrierten Sicherheitsschlüssel (der Schlüssel war an die alte RP ID gebunden). Behandeln Sie es als einmalige Entscheidung und pinnen Sie es auf Ihren stabilen Produktiv-Hostnamen.
- Nutzen Sie *nicht* die Localhost-RP-ID in Produktion. WebAuthn weigert sich, gegen `localhost` von einem nicht-sicheren Kontext zu registrieren.
- Für die Reverse-Proxy-Einrichtung, die diese URLs erst funktionieren lässt, siehe [Reverse-Proxy und TLS konfigurieren](../configure-reverse-proxy-and-tls/).
