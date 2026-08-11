---
title: Budgetposten verwalten
weight: 9
---

Sie wollen eine Einnahme oder Ausgabe erfassen, die nicht automatisch aus Verträgen berechnet wird (Elternbeiträge, Miete, Gartenpflege, Spenden) — und die Zahlen aktuell halten, wenn die Miete steigt, eine Kategorie endet oder eine Einmalzahlung anfällt.

## Budgetposten anlegen

{{< screenshot src="/images/screenshots/budget-items.png" alt="Budgetposten-Liste" caption="Die Budgetposten-Liste zeigt jeden Posten mit aktuellem Monatsbetrag und Kategorie." >}}

1. **Budgetposten** in der Seitenleiste klicken.
2. **Erstellen** klicken.
3. **Namen** setzen (z. B. „Miete“ oder „Elternbeiträge“) und **Einnahme** oder **Ausgabe** wählen.
4. **Speichern**.

## Einträge zum Posten hinzufügen

Jeder Posten hält einen oder mehrere zeitlich begrenzte Einträge. Ein Eintrag bedeutet „dieser Posten kostet/bringt X EUR/Monat zwischen Datum A und Datum B“.

{{< screenshot src="/images/screenshots/budget-item-detail.png" alt="Budgetposten-Detail mit Einträgen" caption="Ein Budgetposten mit mehreren Einträgen — jeder ein zeitlich begrenzter EUR/Monat-Betrag." >}}

1. Den Posten aus der Liste öffnen.
2. **Eintrag hinzufügen** klicken.
3. Setzen:
   - **Von** — Startdatum.
   - **Bis** — Enddatum (leer lassen für laufend).
   - **Betrag** — Eurobetrag **pro Monat**.
   - **Notizen** — optionaler Kontext (z. B. „inkl. Nebenkosten“, „Jahres-Spende auf 12 Monate verteilt“).
4. **Speichern**.

{{< screenshot src="/images/screenshots/budget-item-entry-add.png" alt="Dialog Eintrag hinzufügen" caption="Zeitraum, monatlichen Betrag und optionale Notizen setzen." >}}

Die Einträge fließen in die **Finanzübersicht** und die **Prognose** ein, damit das kumulierte Saldo und die Projektion sie berücksichtigen.

## Budgetposten aktuell halten

Reale Budgets ändern sich. Das passende Bearbeitungsmuster hängt davon ab, was sich ändert:

### Ein Betrag ändert sich unterjährig (Miete steigt, Beitragssatz wird neu verhandelt)

1. Posten öffnen. Der aktive Eintrag ist der mit leerem **Bis**-Feld oder einem **Bis**-Datum in der Zukunft.
2. Auf den **Stift** an diesem Eintrag klicken. **Bis** auf den Tag vor dem Wechsel setzen (neue Miete ab 1. Mai → Bis = 30. April). Speichern.
3. **Eintrag hinzufügen** klicken. **Von** auf das Stichtag-Datum setzen, **Bis** offen lassen, den neuen Monatsbetrag eintragen. Speichern.

Die Detailseite zeigt nun zwei aufeinanderfolgende Einträge. Vergangene Monate nutzen weiterhin den alten Betrag; zukünftige nutzen den neuen. Die Finanzübersicht wechselt am Stichtag automatisch.

### Eine Kategorie endet (Elterngruppe stellt ein, Pacht läuft aus)

1. Posten öffnen, auf den Stift am aktiven Eintrag klicken.
2. **Bis** auf den letzten Tag setzen, an dem der Betrag gilt. Speichern.

Der Posten bleibt ohne aktiven Eintrag bestehen; historische Berichte bleiben korrekt. **Posten nicht löschen** — das Löschen entfernt auch die historischen Einträge.

### Eine Einmalzahlung kommt (Renovierungsrechnung, Anschaffung)

Falls noch kein Posten dafür existiert, einen anlegen (Kategorie *Ausgabe*) und einen einzelnen Eintrag mit **Von** und **Bis** im selben Monat anlegen (oder über den Zeitraum, in dem die Kosten verbucht werden sollen). Notizen helfen dem zukünftigen Sie zu erinnern, worum es ging.

### Falschen Betrag korrigieren (Eingabefehler)

Wenn Sie 200 eingegeben haben, gemeint waren 2000, klicken Sie auf den Stift und ändern Sie den Betrag am bestehenden Eintrag — das ist Korrektur der Historie, also ist Bearbeiten an Ort und Stelle die richtige Aktion.

## Hinweise

- Wiederkehrende monatliche Kosten (Miete): ein Eintrag, der das ganze Jahr abdeckt.
- Einmalige Kosten (Renovierung): ein Eintrag mit kurzem Datumsbereich.
- Jährliche Pauschalbeträge (Spende): über die relevanten Monate verteilen, indem Sie einen kurzen Datumsbereich passend zum Zahlungstermin setzen.
- Beträge werden in **Euro** eingegeben, nicht in Cents. KitaManager konvertiert intern zu Cents — siehe [Warum Geldbeträge als Cents gespeichert werden](../../../explanation/why-money-is-stored-as-cents/).
- Das Protokoll erfasst jede Anlage / Bearbeitung / Löschung eines Eintrags mit Alt → Neu-Werten. Admins können es einsehen: [Protokoll prüfen](../../administer/review-audit-log/).
