---
title: Arbeitsvertrag bei einer Änderung aktualisieren
weight: 18
---

Die Beschäftigungsbedingungen einer Mitarbeiter:in ändern sich: mehr (oder weniger) Wochenstunden, eine andere Entgeltgruppe, ein anderer Bereich oder das Ende eines befristeten Vertrags. Sie wollen die Änderung erfassen, damit Gehaltskosten und Personalabdeckung korrekt bleiben.

Für Stufenaufstiege gibt es eine eigene Anleitung: [Stufenaufstieg dokumentieren](../promote-employee-step/) — diese führt durch das Dashboard-Widget.

## Die Regel: Historie erweitern, nicht überschreiben

Eine reale Änderung hat ein Stichtag-Datum — den Tag, ab dem die neuen Stunden / Entgeltgruppe / der neue Bereich gelten. **Beenden Sie den aktuellen Vertrag am Tag davor und legen Sie einen neuen Vertrag ab dem Stichtag an.** Der alte Vertrag dokumentiert die vorherige Phase; KitaManager nutzt beide für historische Auswertungen, Personalabdeckungen vergangener Monate und das Gehaltsdiagramm.

Wenn Sie stattdessen den bestehenden Vertrag bearbeiten, überschreiben Sie die Historie: eine Gehaltszahl aus drei Monaten zuvor ändert sich stillschweigend, und der Eintrag im Protokoll zeigt nur den „neuen“ Zustand, nicht die echte Änderung.

Ausnahme: **Eingabefehler korrigieren** (Sie haben `30h` statt `35h` von Anfang an eingetragen, das Startdatum war ein Tippfehler). In diesen Fällen den Vertrag direkt bearbeiten — die Zeitleiste spiegelt dann die Realität.

## Schritte — Änderung zum Stichtag erfassen

Diesen Weg nutzen für Stundenänderung, Entgeltgruppen-Wechsel, Bereichs-Transfer oder jedes Vertragsfeld mit einem Stichtag.

{{< screenshot src="/images/screenshots/employee-contracts.png" alt="Vertragshistorie einer Mitarbeiter:in" caption="Die Vertragshistorie mit Aktionen pro Zeile rechts." >}}

1. Öffnen Sie die Mitarbeiter:in über die **Mitarbeiter**-Liste und klicken Sie auf das **Verlaufs**-Symbol, um die Vertragshistorie zu öffnen.
2. Suchen Sie den **aktiven** Vertrag (Status-Badge: *aktiv*). Klicken Sie auf den **Stift** zum Bearbeiten.
3. Setzen Sie **Bis** auf den Tag vor dem Stichtag (z. B. Änderung ab 1. März → Bis = 28. Februar). **Speichern**.
4. Zurück in der Vertragshistorie klicken Sie auf **Neuer Vertrag**.
5. Setzen Sie **Von** auf das Stichtag-Datum. Übernehmen Sie alle Felder vom alten Vertrag und ändern Sie nur das tatsächlich geänderte (Stunden, Entgeltgruppe etc.).
6. **Speichern**.

{{< screenshot src="/images/screenshots/employee-contract-create.png" alt="Dialog Neuer Arbeitsvertrag" caption="Derselbe Dialog wird für Anlegen und Bearbeiten genutzt — nur der Titel ändert sich." >}}

Die Personalstunden- und Gehaltskosten-Werte im Dashboard aktualisieren sich sofort. Ab dem Stichtag fließen die neuen Werte in Personalabdeckung, Finanzübersicht und Prognose ein.

## Sonderfall: Mitarbeiter:in in einen anderen Bereich verschieben

Für einen Bereichswechsel ist Drag-and-Drop auf der **Bereiche**-Seite der schnellste Weg — bei einem Vertrag, der vor heute begann, schließt KitaManager den alten Vertrag und legt einen neuen im Zielbereich automatisch an. Der Vertrag-Bearbeiten-Dialog selbst hat kein Bereichs-Feld, deshalb können Sie den Bereich von dieser Seite aus nicht ändern. Beide Wege (Drag-and-Drop und Planung im Voraus) stehen in [Mitarbeiter:in zwischen Bereichen verschieben](../move-employee-between-sections/).

## Sonderfall: Vertragsende (befristet, Austritt, Elternzeit)

Der Vertrag endet und es folgt nicht unmittelbar ein neuer. Setzen Sie das **Bis**-Datum auf dem aktiven Vertrag und speichern — kein Folgevertrag nötig. Ab dem Tag nach **Bis** trägt die Mitarbeiter:in nicht mehr zur Personalanforderung und zu den Gehaltskosten bei.

Der Mitarbeiter-Eintrag bleibt ohne aktiven Vertrag bestehen — das ist korrekt: historische Berichte referenzieren weiterhin den alten Vertrag.

## Falsches Datum korrigieren (kein neuer Vertrag nötig)

Wenn nur das Start- oder Enddatum falsch ist (Sie haben den 1. März eingetragen, der Vertrag begann tatsächlich am 15. März), geht eines von beiden:

- **Bearbeiten-Dialog** — Vertrag öffnen, **Von** / **Bis** ändern, speichern.
- **Zeitleisten-Ansicht** — Tab **Zeitleiste** auf der Vertragsseite öffnen und die Vertragsgrenze auf das richtige Datum ziehen. Hilfreich, wenn man alle Verträge auf einen Blick sehen will.

## Hinweise

- Stufenaufstiege haben ein eigenes Widget und eine eigene Anleitung — folgen Sie *nicht* dem manuellen Weg hier. Siehe [Stufenaufstieg dokumentieren](../promote-employee-step/).
- Eine falsche Kombination aus Entgeltgruppe/Stufe/Stunden berechnet die Gehaltskosten still verfälscht. Prüfen Sie den neuen Vertrag am Gehaltsdiagramm (das Diagramm aktualisiert sich sofort nach dem Speichern).
- Das Protokoll erfasst jede Vertragsänderung **mit dem alten und dem neuen Wert** der geänderten Felder — Datum, Bereich, Entgeltgruppe, Stufe, Wochenstunden und Entgelttabelle. Admins können es einsehen: [Protokoll prüfen](../../administer/review-audit-log/).
- Zur Berechnungskette für Gehälter (Entgeltgruppe × Stufe × Stunden × Tabelle), siehe die Admin-Anleitung [Entgelttabelle aktualisieren](../../administer/update-pay-plan/).
