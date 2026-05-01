---
title: ISBJ-Abrechnung hochladen
weight: 6
---

Sie wollen Ihren monatlichen ISBJ-Bescheid (die Excel-Datei vom Senat) hochladen, damit KitaManager sie mit der eigenen Förderberechnung abgleicht.

## Schritte

1. Klicken Sie in der Seitenleiste auf **Abrechnungen**.
2. Klicken Sie auf **Hochladen** und wählen Sie die Excel-Datei vom Computer.
3. Die Abrechnung erscheint in der Liste, gruppiert nach Kita-Jahr.
4. Die Übersicht zeigt sofort, wie viele Kinder gematcht haben, wie viele Differenzen aufweisen und die monetäre Gesamtdifferenz.
5. Klicken Sie auf die Abrechnungs-Zeile, um den pro-Kind-Vergleich zu öffnen.

## Was im Hintergrund passiert

KitaManager parst die Excel, normalisiert sie zu pro-Kind-Einträgen und joint jeden Eintrag gegen Ihre Betreuungsverträge über die Gutscheinnummer. Für jeden Match werden Beträge verglichen; für jeden Nicht-Match wird als **Fehlt in Abrechnung** oder **Zusätzlich in Abrechnung** kategorisiert. Siehe [Der ISBJ-Abgleich](../../../explanation/the-isbj-reconciliation-flow/) für die vollständige Pipeline.

## Hinweise

- Erneutes Hochladen derselben Monats-Abrechnung ersetzt die vorherige — die alten pro-Kind-Zeilen werden gelöscht und die neuen eingefügt. Der Audit-Log dokumentiert die Ersetzung.
- Abrechnungen sind organisations-bezogen. Jede Kita lädt ihre eigenen hoch.
- Abweichungen lösen sich nicht von selbst. Nach dem Hochladen ist der nächste Schritt [Abweichung in einer Abrechnung untersuchen](../investigate-a-bill-discrepancy/).
