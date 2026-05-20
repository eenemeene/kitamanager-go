---
title: Personendaten eines Kindes oder einer Mitarbeiter:in aktualisieren
weight: 21
---

Sie wollen Personendaten korrigieren oder aktualisieren — Vorname, Nachname, Geschlecht oder Geburtsdatum. Vielleicht hat eine Familie eine Heiratsurkunde abgegeben, eine Kolleg:in den Namen geändert oder Sie haben einen Tippfehler beim Geburtsdatum mit dem Kita-Gutschein bemerkt.

Der Personendaten-Bearbeiten-Dialog ist für Kinder und Mitarbeiter:innen identisch. Er bearbeitet nur diese vier Felder; Verträge, Gutscheine, Anwesenheit und Abrechnungshistorie sind nicht betroffen.

## Personendaten eines Kindes aktualisieren

{{< screenshot src="/images/screenshots/children.png" alt="Kinderliste" caption="In jeder Zeile gibt es ein Stift-Symbol — Personendaten bearbeiten." >}}

1. **Kinder**-Liste öffnen.
2. Kind suchen und auf das **Stift**-Symbol in der Zeile klicken.
3. Die benötigten Felder ändern:
   - **Vorname** / **Nachname** — exakt wie auf dem Kita-Gutschein geschrieben.
   - **Geschlecht** — wird ausschließlich für Statistiken genutzt, nicht für die Förderung.
   - **Geburtsdatum** — den Hinweis unten lesen, bevor Sie das ändern.
4. **Speichern**.

{{< screenshot src="/images/screenshots/child-edit-personal.png" alt="Kind-Personendaten bearbeiten" caption="Der Personendaten-Dialog — vier Felder, nichts weiter." >}}

## Personendaten einer Mitarbeiter:in aktualisieren

Derselbe Dialog, dieselben Felder, geöffnet aus der **Mitarbeiter**-Liste über das Stift-Symbol. Der Dialog-Titel ändert sich zu *Mitarbeiter:in bearbeiten*; alles andere ist identisch.

{{< screenshot src="/images/screenshots/employees.png" alt="Mitarbeiterliste" caption="Stift pro Zeile — derselbe Dialog wie bei Kindern." >}}

## Achtung — Korrekturen am Geburtsdatum verändern die Förderung still

Das Geburtsdatum des Kindes bestimmt, welcher Altersbereichs-Eintrag aus der Förderkonfiguration angewendet wird (siehe [Wie Vertragseigenschaften die Förderung bestimmen](../../../explanation/how-contract-properties-determine-funding/)). Ein Wechsel etwa von *2022-07-15* auf *2022-08-15* kann das Kind in einem anderen Monat von einer Altersgruppe in die nächste verschieben und damit die berechnete Förderung für jeden Bescheid-Monat ab Vertragsbeginn verändern.

Vor dem Speichern einer Geburtsdatums-Änderung:

- Mit dem Kita-Gutschein-Papier abgleichen. Der Abgleich-Algorithmus für ISBJ sieht nur Monat und Jahr, ein kleiner Tippfehler beim Tag bricht den Abgleich meist nicht — ein falscher Monat oder ein falsches Jahr schon.
- Prüfen, ob das Kind nun nahe an der Schuleinschulungs-Grenze („Muss-Kind") für Ihr Bundesland liegt. KitaManager füllt das Vertragsende-Datum aus Geburtsdatum + Bundesland automatisch aus, wenn ein neuer Vertrag angelegt wird; **bereits angelegte Verträge haben weiterhin das alte Enddatum** — diese durchsehen und ggf. anpassen.

Bei einer Mitarbeiter:in beeinflusst das Geburtsdatum nur die Anzeige — Gehalt und Personalplanung sind nicht betroffen.

## Achtung — Namensänderungen können den ISBJ-Auto-Abgleich brechen

Wenn der nächste ISBJ-Bescheid „Müller, Maria" enthält und Sie den Eintrag in „Mueller, Maria" umbenannt haben, fällt der Auto-Abgleich auf die Gutscheinnummer zurück. Das Dashboard meldet das als *Namens-Vorschlag* auf der Karte **Kinder ohne Gutschein** oder im Bescheid-Vergleich — Vorschlag annehmen, um die Namen wieder anzugleichen.

Vermeiden Sie kreative Schreibweisen. Spiegeln Sie das Kita-Gutschein-Papier exakt, inkl. Bindestriche, Umlaute und Doppelnamen.

## Was dieser Dialog NICHT ändert

- **Verträge** — Betreuungsart, Zuschläge, Stunden, Bereich, Daten. Siehe [Betreuungsvertrag eines Kindes aktualisieren](../update-child-contract/) oder [Arbeitsvertrag aktualisieren](../update-employee-contract/).
- **Gutscheine** — siehe [Kita-Gutschein-Nummer zuweisen](../assign-a-voucher/).
- **Anwesenheit** — historische Anwesenheitseinträge werden nicht umgeschrieben; der neue Name erscheint ab sofort im Anwesenheits-Raster.

## Hinweise

- Das Protokoll erfasst jede Personendaten-Änderung mit Alt → Neu-Wert. Admins können es einsehen: [Protokoll prüfen](../../administer/review-audit-log/).
- Eintrag löschen und neu anlegen ist **keine** Alternative — Löschen entfernt Anwesenheits-, Vertrags- und Abrechnungshistorie. Immer in-place bearbeiten.
- Geschlecht ist eine geschlossene Liste (männlich, weiblich, divers), passend zu den deutschen Standesamts-Optionen.
