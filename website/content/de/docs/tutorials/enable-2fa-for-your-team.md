---
title: Zwei-Faktor-Authentifizierung im Team einführen
weight: 3
---

Dieses Tutorial begleitet Sie dabei, 2FA für jedes Konto in Ihrer KitaManager-Organisation, das Daten bearbeiten kann, zum Standard zu machen. Am Ende hat jedes Admin- und Manager-Konto TOTP eingerichtet, und Sie haben einen dokumentierten Prozess für neue Kolleg:innen.

Sie brauchen:

- Ein Admin-Konto in Ihrer Organisation.
- Etwa 15 Minuten Ihrer eigenen Zeit, plus 5 Minuten pro Kolleg:in (das meiste machen sie selbst).

## Schritt 1 — Erst auf Ihrem eigenen Konto 2FA aktivieren

Mit gutem Beispiel vorangehen. Gehen Sie [Zwei-Faktor-Authentifizierung aktivieren](../../how-to/use/enable-2fa/) auf Ihrem eigenen Konto Ende-zu-Ende durch — inklusive:

- QR-Code mit der Authenticator-App scannen, die Sie empfehlen werden (Google Authenticator, 1Password, Authy, Bitwarden — eine wählen und standardisieren).
- Wiederherstellungscodes irgendwo speichern, wo Sie sie in 18 Monaten wiederfinden.
- Optional einen Sicherheitsschlüssel hinzufügen.

Abmelden und wieder anmelden, um zu prüfen, dass die Aufforderung wie erwartet funktioniert.

## Schritt 2 — Rollout planen

Für jede:n Kolleg:in brauchen Sie:

- Deren Arbeits-E-Mail und -Passwort (haben sie bereits).
- Ein kurzes Zeitfenster, in dem sie an einem Bildschirm und nicht in Eile sind.
- Einen Kommunikationskanal: in Person, Videoanruf, Slack-Direktnachricht. **Keine E-Mail.**

Entscheiden Sie:

- **Welche Authenticator-App** das Team nutzt. Standardisierung erlaubt eine einheitliche Anleitung und einen einheitlichen Trouble-shooting-Pfad.
- **Wo Wiederherstellungscodes hinkommen.** Optionen: jede:r speichert die eigenen im persönlichen Passwort-Manager (am besten); oder ausgedruckt in einer abschließbaren Schublade (ok); oder als gemeinsamer Passwort-Tresor-Eintrag pro Person (akzeptabel, aber von anderen einsehbar).
- **Was „Authenticator und Codes verloren" bedeutet.** Heute ist die einzige Wiederherstellung das Löschen und Neuanlegen des Kontos. Sagen Sie das den Nutzer:innen, damit sie die Codes ernst nehmen.

## Schritt 3 — Jede:n Kolleg:in einrichten

Für jede:n:

1. Setzen Sie sich mit ihnen (persönlich oder per Bildschirmfreigabe) auf **Einstellungen → Zwei-Faktor-Authentifizierung**.
2. Gehen Sie [Zwei-Faktor-Authentifizierung aktivieren](../../how-to/use/enable-2fa/) Schritt für Schritt durch.
3. **Verifizieren Sie gemeinsam**, dass die Wiederherstellungscodes irgendwo gespeichert sind, wo sie sie wiederfinden. Beobachten Sie, wie sie gespeichert werden. Wenn sie sagen „mach ich später", bleiben Sie dabei, bis es passiert ist.
4. Melden Sie sie ab und wieder an, um zu bestätigen, dass die nächste Anmeldung den Code abfragt.
5. Notieren Sie das Einrichtungsdatum (z. B. im Team-Wiki), damit Sie eine Aufzeichnung haben.

## Schritt 4 — Prozess für neue Mitglieder dokumentieren

Erweitern Sie Ihr Onboarding für neue Admins/Manager:

> Tag 1, nach erstem Login:
> 1. Initiales Passwort ändern.
> 2. 2FA aktivieren nach [Zwei-Faktor-Authentifizierung aktivieren](../../how-to/use/enable-2fa/).
> 3. Wiederherstellungscodes im Passwort-Manager speichern.
> 4. Mit Team-Lead bestätigen.

## Schritt 5 — Regelmäßige Kontrolle

Einmal pro Quartal Ihre **Nutzer:innen**-Liste auf jemanden mit Bearbeitungsrechten ohne aktivierten Faktor durchsehen. Es gibt noch keinen „Nutzer:innen ohne 2FA"-Bericht — manuelles Durchsehen von `/api/v1/users/{id}/factors` für jede:n Admin/Manager genügt.

Wenn Sie ein verwaistes Konto finden (eingerichtet, aber Nutzer:in hat keinen Zugriff mehr auf das Gerät), setzen Sie deren Passwort zurück, damit sie sich neu anmelden, und gehen Sie die Einrichtung erneut mit ihnen durch.

## Sie sind fertig

Die Konten Ihres Teams sind durch etwas Geschütztes (das Gerät), zusätzlich zu etwas Bekanntem (das Passwort). Ein geleaktes Passwort allein reicht nicht mehr, um die Daten Ihrer Kita zu kompromittieren.

Als nächstes:

- Stellen Sie sicher, dass Backups laufen — siehe [Datenbank sichern und wiederherstellen](../../how-to/operate/back-up-and-restore/).
- Prüfen Sie regelmäßig den [Audit-Log](../../how-to/administer/review-audit-log/) auf Auffälligkeiten.
- Halten Sie die [Fördersätze](../../how-to/operate/update-government-funding-rates/) aktuell, damit der Abgleich akkurat bleibt.
