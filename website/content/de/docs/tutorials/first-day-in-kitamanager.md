---
title: Erster Tag in KitaManager
weight: 1
aliases:
  - /docs/user-guide/
---

Dieses Tutorial bringt Sie in etwa 30 Minuten von „Ich habe gerade einen Zugang bekommen“ zu „Ich kann KitaManager im Kita-Alltag souverän nutzen“. Sie werden sich anmelden, das Dashboard erkunden, einen Anwesenheitstag erfassen, einen Vertrag eines Kindes ansehen, eine fiktive ISBJ-Abrechnung hochladen und Ihren ersten Bericht lesen.

Sie brauchen:

- Eine laufende KitaManager-Instanz — falls noch nicht vorhanden, machen Sie zuerst [KitaManager bereitstellen](../deploy-kitamanager/).
- Ein Admin- oder Manager-Konto in einer Organisation, in die die **Kita Sonnenschein**-Beispieldaten geladen sind (jede lokale Dev-Installation hat sie standardmäßig).
- Etwa 30 Minuten Zeit.

Wenn Sie fertig sind, sollten Sie sich ohne Doku zurechtfinden und wissen, in welche [How-to-Anleitung](../../how-to/use/) Sie für eine konkrete Aufgabe schauen.

## Schritt 1 — Anmelden

1. Öffnen Sie KitaManager im Browser.
2. Geben Sie E-Mail-Adresse und Passwort ein und klicken Sie **Anmelden**.

Wenn Ihr Konto Zwei-Faktor-Authentifizierung aktiviert hat, geben Sie den Code aus Ihrer Authenticator-App ein oder tippen Sie auf Ihren Sicherheitsschlüssel. Falls Sie 2FA noch nie eingerichtet haben: das Rezept [Zwei-Faktor-Authentifizierung aktivieren](../../how-to/use/enable-2fa/) führt Sie durch — empfohlen für jedes Konto, das Daten bearbeiten kann.

Nach der Anmeldung landen Sie auf dem **Dashboard**. Es ist das Herz von KitaManager: hier erscheinen alle wichtigen Hinweise.

## Schritt 2 — Das Dashboard durchgehen

Das Dashboard ist von oben nach unten nach Dringlichkeit organisiert: Kennzahlen-Karten (KPIs für den aktuellen Monat), dann Warn-Karten, die nur erscheinen, wenn etwas zu tun ist, dann Routine-Widgets für anstehende Arbeit. Hovern Sie über einen Karten-Titel für eine ein-zeilige Erklärung.

**Aktion:** Lesen Sie die Beschreibungen auf jeder Karte zwei Minuten lang. Wenn Ihr Dashboard Warnungen anzeigt (die Beispieldaten der Kita Sonnenschein haben einige Vertragsabweichungen), haben Sie Ihre erste Aufgabe gefunden.

## Schritt 3 — Einen Anwesenheitstag erfassen

1. Klicken Sie in der Seitenleiste auf **Anwesenheit**.
2. Sie sehen ein Wochenraster: Zeilen sind Kinder mit aktivem Betreuungsvertrag, Spalten sind Wochentage.
3. Tippen (oder klicken) Sie eine Zelle, um das Kind als anwesend, abwesend oder leer zu markieren. Speichern erfolgt automatisch — keine Speichern-Schaltfläche.
4. Mit den Pfeilen oben wechseln Sie zwischen Wochen.

Die Übersicht oben zeigt Ihnen für die aktuelle Woche, wie viele Kinder pro Tag anwesend waren. Wenn Sie die Anwesenheitshistorie eines einzelnen Kindes sehen möchten, öffnen Sie dessen Detailseite; das Anwesenheits-Widget ist die breite Sicht.

## Schritt 4 — Einen Betreuungsvertrag eines Kindes ansehen

1. Klicken Sie in der Seitenleiste auf **Kinder**.
2. Die Liste zeigt alle aufgenommenen Kinder mit aktuellem Förderbetrag und Abrechnungsdifferenzen. Wählen Sie ein Kind, das Ihnen ins Auge fällt, und klicken Sie es an.
3. Scrollen Sie zum Abschnitt **Verträge**. Sie sehen einen oder mehrere Betreuungsverträge — jeder hat ein Von-/Bis-Datum, einen Bereich, eine Gutscheinnummer, eine Betreuungsart und alle Zuschläge (NdH, QM/MSS, Integration).

Beim Hover über eine Vertragszeile sehen Sie die berechnete monatliche Förderung. Wenn Sie die Zuschläge nicht kennen, ist [Wie die Förderung in Berlin funktioniert](../../explanation/how-funding-works-in-berlin/) die maßgebliche Erklärung — einmal lesen, danach müssen Sie die Abkürzungen nie wieder nachschlagen.

**Aktion:** Vergleichen Sie, was in KitaManager steht, mit Ihrem letzten Papier-Bescheid. Gleiche Gutscheinnummer? Gleiche Betreuungsart? Gleiche Zuschläge? Wenn ja, sollte dieses Kind abrechnen; wenn nein, haben Sie eine Korrektur identifiziert.

## Schritt 5 — Einen ISBJ-Abrechnungs-Vergleich öffnen

Die Beispieldaten enthalten bereits einen ISBJ-Bescheid, sodass Sie für den Vergleichs-Workflow keine echte Excel brauchen.

1. Klicken Sie in der Seitenleiste auf **Abrechnungen**.
2. Klicken Sie auf die neueste Bescheid-Zeile.
3. Die Detailansicht zeigt einen pro-Kind-Vergleich: jede Zeile ein Kind, zwei Spalten für KitaManagers berechneten und den ISBJ-Betrag, plus Status (Übereinstimmung / abweichend / fehlt in Abrechnung / zusätzlich in Abrechnung).

Die geseedeten Daten enthalten absichtlich einige Abweichungen. Klicken Sie auf eine rote Zeile, um zu sehen, welche Vertragseigenschaft abweicht. Der nächste Schritt wäre [Abweichung in einer Abrechnung untersuchen](../../how-to/use/investigate-a-bill-discrepancy/) — für das Tutorial reicht es, dass Sie den Vergleich lesen können.

(Für das Hochladen Ihres eigenen monatlichen Bescheids siehe [Senatsabrechnung hochladen](../../how-to/use/upload-an-isbj-bill/).)

## Schritt 6 — Ihren ersten Bericht lesen

Klicken Sie in der Seitenleiste auf **Statistiken** und wählen Sie **Personalstunden**.

Das Personalstunden-Diagramm zeigt zwei Linien: wie viele Personalstunden Ihre Kinder benötigen (berechnet aus den pro-Kind-VZÄ-Bedarfen der Förder-Konfiguration) und wie viele Sie tatsächlich haben (aus aktiven Arbeitsverträgen). Wenn die verfügbare Linie unter der benötigten liegt, sind Sie in dem Zeitraum unterbesetzt.

Für eine vollständige Tour durch jedes Diagramm siehe [Bericht drucken](../../how-to/use/print-a-report/) — Sie können jeden Bericht ausdrucken und mit zur Vorstandssitzung nehmen.

## Sie sind fertig

Sie haben sich angemeldet, die Struktur des Dashboards verstanden, Anwesenheit erfasst, einen Vertrag inspiziert, eine Abrechnung hochgeladen und einen Bericht gelesen. Die alltägliche Form von KitaManager ist Ihnen jetzt vertraut.

Wohin als nächstes:

- Für eine konkrete Aufgabe springen Sie zur passenden [How-to-Anleitung](../../how-to/use/).
- Um zu verstehen, *warum* die Förderung so funktioniert, ist [Wie die Förderung in Berlin funktioniert](../../explanation/how-funding-works-in-berlin/) die wertvollsten 15 Minuten Lesezeit.
- Wenn Sie Admin sind und einen Kollegen-Account anlegen müssen, siehe [Nutzer:in anlegen](../../how-to/administer/create-a-user/).
