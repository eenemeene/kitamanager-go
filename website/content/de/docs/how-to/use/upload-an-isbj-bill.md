---
title: Senatsabrechnung hochladen
weight: 6
---

Jeden Monat schickt der Senat eine Excel-Datei mit den Förderbeträgen für Ihre Kinder — die **Senatsabrechnung** (im ISBJ-Verfahren auch *ISBJ-Bescheid* genannt). Sie laden sie hoch, damit KitaManager sie mit der eigenen Berechnung vergleicht.

## Schritte

{{< screenshot src="/images/screenshots/government-funding-bills.png" alt="Seite Senatsabrechnungen mit dem Auswahlfeld für die Excel-Datei" caption="Das Feld **ISBJ Excel-Datei auswählen (.xlsx)** liegt oben auf der Seite. Der Button **Hochladen** wird erst aktiv, wenn eine Datei gewählt ist." >}}

1. Klicken Sie in der Seitenleiste auf **Abrechnungen**.
2. Wählen Sie im Feld **ISBJ Excel-Datei auswählen (.xlsx)** die Datei von Ihrem Computer aus.
3. Klicken Sie auf **Hochladen**. Der Button bleibt ausgegraut, solange keine Datei gewählt ist.
4. Die Abrechnung erscheint in der Tabelle darunter, nach Kita-Jahr gruppiert.
5. Die Leiste über der Tabelle zeigt sofort: wie viele Abrechnungen stimmen, wie viele Abweichungen haben, und die Gesamtdifferenz in Euro.
6. Klicken Sie in der Spalte **Aktionen** auf das **Augen-Symbol**, um den Vergleich Kind für Kind zu öffnen.

## Was KitaManager dabei tut

KitaManager liest die Excel-Datei und ordnet jede Zeile über die Gutscheinnummer einem Betreuungsvertrag zu. Für jedes zugeordnete Kind vergleicht es die Beträge.

Was sich nicht zuordnen lässt, erscheint in einer von zwei Gruppen:

- **Fehlt in Abrechnung** — KitaManager kennt das Kind, die Abrechnung führt es nicht.
- **Zusätzlich in Abrechnung** — die Abrechnung führt ein Kind, das KitaManager nicht kennt.

Den ganzen Ablauf beschreibt [Der ISBJ-Abgleich](../../../explanation/the-isbj-reconciliation-flow/).

## Hinweise

- Laden Sie dieselbe Monatsabrechnung erneut hoch, ersetzt sie die vorherige: die alten Zeilen werden gelöscht, die neuen eingefügt. Das Protokoll hält die Ersetzung fest.
- Jede Kita lädt ihre eigene Abrechnung hoch — eine Abrechnung gehört immer zu genau einer Organisation.
- Abweichungen lösen sich nicht von selbst. Der nächste Schritt ist [Abweichung in einer Abrechnung untersuchen](../investigate-a-bill-discrepancy/).
