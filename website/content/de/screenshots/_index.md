---
title: Bildschirmfotos
weight: 3
---

Ein Rundgang durch die KitaManager Go Benutzeroberfläche mit den wichtigsten Bildschirmen für den täglichen Einsatz.

---

## Anmeldung

Der Anmeldebildschirm ist der Einstiegspunkt in KitaManager. Benutzer authentifizieren sich mit ihrer E-Mail-Adresse und ihrem Passwort. Nach erfolgreicher Anmeldung wird ein JWT-Token ausgestellt, der die Sitzung aktiv hält.

{{< screenshot src="/images/screenshots/login.png" alt="Anmeldeseite" caption="Die Anmeldeseite — geben Sie E-Mail und Passwort ein, um auf das System zuzugreifen." >}}

---

## Dashboard

Nach der Anmeldung bietet das Dashboard einen Überblick über Ihre Kita auf einen Blick. Die obere Zeile zeigt Zusammenfassungskarten für die Gesamtzahl der Organisationen, Mitarbeiter, Kinder und Benutzer. Darunter zeigen die **Schnellstatistiken** Details für die aktuell ausgewählte Organisation.

Die linke Seitenleiste ist Ihre Hauptnavigation — sie ist unterteilt in systemweite Einträge (Dashboard, Organisationen, Landesförderungen) und organisationsbezogene Einträge (Benutzer, Gruppen, Mitarbeiter, Kinder, Statistiken, Vergütungspläne). Über das Dropdown in der Seitenleiste können Sie zwischen Organisationen wechseln.

{{< screenshot src="/images/screenshots/dashboard.png" alt="Dashboard" caption="Das Dashboard mit Zusammenfassungskarten und organisationsbezogenen Schnellstatistiken." >}}

---

## Organisationen

Die Organisationsseite listet alle Kita-Einrichtungen auf, auf die Sie Zugriff haben. Jede Zeile zeigt den Organisationsnamen, das Bundesland und ob die Organisation aktuell aktiv ist. Von hier aus können Sie neue Organisationen erstellen oder bestehende bearbeiten.

Wenn Sie mehrere Kitas verwalten, erhalten Sie hier das Gesamtbild über alle Ihre Einrichtungen.

{{< screenshot src="/images/screenshots/organizations.png" alt="Organisationsliste" caption="Organisationsübersicht — verwalten Sie mehrere Kita-Einrichtungen an einem Ort." >}}

---

## Mitarbeiter

Die Mitarbeiterliste zeigt alle Beschäftigten der ausgewählten Organisation. Sie sehen auf einen Blick Name, Geschlecht, Geburtsdatum, Alter, aktuelle Position, Entgeltgruppe und -stufe sowie Wochenstunden jedes Mitarbeiters. Die Aktionsschaltflächen rechts ermöglichen es, die Vertragshistorie einzusehen, Details anzuzeigen, den Mitarbeiterdatensatz zu bearbeiten oder einen Mitarbeiter zu entfernen.

Neue Mitarbeiter können über die Schaltfläche **+ New Employee** oben rechts hinzugefügt werden.

{{< screenshot src="/images/screenshots/employees.png" alt="Mitarbeiterliste" caption="Mitarbeiterübersicht mit persönlichen Daten, Position, Entgeltgruppe und Wochenstunden." >}}

---

## Kinder

Die Kinderliste zeigt jedes angemeldete Kind in der ausgewählten Organisation. Jede Zeile zeigt Name, Geschlecht, Geburtsdatum, Alter, aktuellen Vertragsstatus, Betreuungseigenschaften (wie halbtag, ganztag, ndh oder integration) und den **automatisch berechneten monatlichen Förderbetrag** basierend auf der aktiven Landesförderungs-Konfiguration.

Dies ist der Bildschirm, den Kita-Leitungen am häufigsten nutzen — er bietet auf einen Blick ein vollständiges Bild von Anmeldungen und Förderung.

{{< screenshot src="/images/screenshots/children.png" alt="Kinderliste" caption="Kinderübersicht mit Anmeldestatus, Betreuungseigenschaften und berechneten Förderbeträgen." >}}

---

## Landesförderung

Die Landesförderungsseite ermöglicht es Administratoren, die landesspezifischen Förderungsregeln zu konfigurieren, die die automatische Förderberechnung steuern. Jeder Eintrag repräsentiert eine Förderungs-Konfiguration für ein bestimmtes Bundesland (z.B. "Berlin Kita-Förderung"). Innerhalb jeder Konfiguration können Sie Zeiträume und eigenschaftsbasierte Förderbeträge definieren.

Wenn die Vertragseigenschaften eines Kindes mit einem Fördereintrag übereinstimmen, wird der entsprechende Monatsbetrag automatisch in der Kinderliste angezeigt.

{{< screenshot src="/images/screenshots/government-funding-rates.png" alt="Landesförderung" caption="Landesförderungs-Konfigurationen — definieren Sie landesspezifische Regeln für automatische Förderberechnungen." >}}
