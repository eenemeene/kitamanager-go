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

Dieselbe Gutscheinnummer erneut auf dasselbe Kind zu schreiben ist ein No-Op. Eine Gutscheinnummer, die bereits an einem anderen Kind hängt, wird mit einem Conflict abgelehnt; vorher beim alten Kind entfernen.

## Hinweise

- Gutscheinnummern sind systemweit eindeutig (entsprechend der Nummerierung des Bezirks-Jugendamts).
- Wenn Sie für vergangene Monate bereits ISBJ-Bescheide hochgeladen haben, ändert sich der Vergleich für diese Monate nicht — die Bescheid-Daten sind beim Upload festgefroren. Laden Sie die betreffenden Bescheide erneut hoch, wenn Sie die Matches rückwirkend brauchen.
- Für den allgemeinen Untersuchungs-Workflow bei nicht-passenden Bescheiden siehe [Abweichung in einer Abrechnung untersuchen](../investigate-a-bill-discrepancy/).
- Für die pro-Kind-Abrechnungssicht (welcher Gutschein welchen Bescheid getroffen hat) auf der Detailseite des Kindes auf **Abrechnungshistorie** klicken.
