---
title: Kita-Gutschein-Nummer zuweisen
weight: 15
---

Ein Kind erscheint auf dem Dashboard unter „Kinder ohne Gutscheinnummer", oder Sie haben einen frischen Kita-Gutschein vom Bezirks-Jugendamt bekommen. Sie wollen die Gutscheinnummer hinterlegen, damit KitaManager dieses Kind beim nächsten ISBJ-Bescheid abgleichen kann.

## Schritte (vom Dashboard)

1. Dashboard öffnen. Die Karte **Kinder ohne Gutscheinnummer** listet jedes Kind mit Vertrag aber ohne Gutscheinnummer.
2. Pro Zeile die Gutscheinnummer vom Papier-Gutschein in das Inline-Eingabefeld einfügen.
3. **Eingabe** drücken (oder das Speichern-Symbol klicken). Das Kind verschwindet sofort aus der Warn-Liste.

Wenn das Dashboard einen Namens-Vorschlag macht (der Bescheid nennt „Müller, Maria", Sie hatten „Maria Mueller" eingegeben), erscheint eine **Vorschlag übernehmen**-Schaltfläche. Damit gleichen Sie die Namen mit einem Klick an.

## Weiteren Gutschein hinzufügen (Verlängerung oder Korrektur)

Gutscheine sind eine Liste pro Kind — ein neuer Gutschein ersetzt nicht den alten; beide bleiben hinterlegt. Neuen Wert über dasselbe Dashboard-Eingabefeld oder über die API hinzufügen (`POST /organizations/{orgId}/children/{childId}/vouchers`). Der neueste Gutschein matcht aktuelle Bescheide; alte Gutscheine bleiben für historische Abgleiche im Datensatz.

Die Zuweisung ist idempotent: erneutes Senden einer schon bekannten Gutscheinnummer ist ein No-Op, kein Fehler.

## Hinweise

- Gutscheinnummern müssen innerhalb einer Organisation eindeutig sein.
- Wenn Sie für vergangene Monate bereits ISBJ-Bescheide hochgeladen haben, ändert sich der Vergleich für diese Monate nicht — die Bescheid-Daten sind beim Upload festgefroren. Laden Sie die betreffenden Bescheide erneut hoch, wenn Sie die Matches rückwirkend brauchen.
- Für den allgemeinen Untersuchungs-Workflow bei nicht-passenden Bescheiden siehe [Abweichung in einer Abrechnung untersuchen](../investigate-a-bill-discrepancy/).
- Für die pro-Kind-Abrechnungssicht (welcher Gutschein welchen Bescheid getroffen hat) auf der Detailseite des Kindes auf **Abrechnungshistorie** klicken.
