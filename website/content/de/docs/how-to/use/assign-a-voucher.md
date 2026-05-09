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

Gutscheine sind eine Liste pro Kind — ein neuer Gutschein ersetzt nicht den alten; beide bleiben hinterlegt.

### Aus der Kinderliste (empfohlen)

1. **Kinder** in der Seitenleiste öffnen.
2. Das Kind suchen und in der Zeile auf die **Voucher-Nummern**-Aktion (Ticket-Symbol) klicken.
3. Der Voucher-Dialog öffnet sich mit allen aktuell hinterlegten Nummern. Neue Kita-Gutschein-Nummer im Format `GB-DDDDDDDDDDD-NN` (11 Ziffern, Bindestrich, 2 Ziffern) eingeben und auf **Hinzufügen** klicken.
4. Um eine falsche Nummer zu entfernen (z. B. Tippfehler-Korrektur), auf das Papierkorb-Symbol neben dem Eintrag klicken und bestätigen. Der eindeutige Slot wird freigegeben, sodass dieselbe Nummer danach einem anderen Kind zugewiesen werden kann.

### Vom Dashboard

Die Karte **Kinder ohne Gutscheinnummer** auf dem Dashboard bleibt der schnellste Weg für die *erste* Zuweisung nach Anlegen eines Kindes oder Hochladen eines Bescheids. Sobald ein Gutschein an einem Kind hängt, den obigen Dialog für weitere Hinzufügungen oder Korrekturen verwenden.

### Über die API

`POST /organizations/{orgId}/children/{childId}/vouchers` (hinzufügen) und `DELETE /organizations/{orgId}/children/{childId}/vouchers/{voucherId}` (entfernen). Der neueste Gutschein matcht aktuelle Bescheide; alte Gutscheine bleiben für historische Abgleiche im Datensatz.

Dieselbe Gutscheinnummer erneut auf dasselbe Kind zu schreiben ist ein No-Op. Eine Gutscheinnummer, die bereits an einem anderen Kind hängt, wird mit einem Conflict abgelehnt; vorher beim alten Kind über den Voucher-Dialog entfernen.

## Wer darf was

- **Gutscheinnummern ansehen**: jede Rolle mit Zugriff auf das Kind (Admin, Manager, Mitglied, Personal).
- **Gutscheinnummern hinzufügen oder entfernen**: nur Admins und Manager. Mitglieder und Personal sehen die Liste schreibgeschützt — das Eingabefeld und die Entfernen-Buttons sind für sie ausgeblendet.

## Hinweise

- Gutscheinnummern sind systemweit eindeutig (entsprechend der Nummerierung des Bezirks-Jugendamts).
- Wenn Sie für vergangene Monate bereits ISBJ-Bescheide hochgeladen haben, ändert sich der Vergleich für diese Monate nicht — die Bescheid-Daten sind beim Upload festgefroren. Laden Sie die betreffenden Bescheide erneut hoch, wenn Sie die Matches rückwirkend brauchen.
- Für den allgemeinen Untersuchungs-Workflow bei nicht-passenden Bescheiden siehe [Abweichung in einer Abrechnung untersuchen](../investigate-a-bill-discrepancy/).
- Für die pro-Kind-Abrechnungssicht (welcher Gutschein welchen Bescheid getroffen hat) auf der Detailseite des Kindes auf **Abrechnungshistorie** klicken.
