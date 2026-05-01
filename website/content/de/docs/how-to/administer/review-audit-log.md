---
title: Audit-Log der Organisation prüfen
weight: 8
---

Sie wollen sehen, wer in Ihrer Organisation was und wann geändert hat.

## Schritte

1. Öffnen Sie in der Seitenleiste **Einstellungen** → **Audit-Log**.
2. Die Tabelle zeigt Zeitpunkt, Nutzer:in, Aktion (z. B. `child_create`, `section_delete`), betroffene Ressource, IP-Adresse und Ergebnis.
3. Filtern Sie nach Datumsbereich oder geben Sie eine Zeichenfolge in **Aktion** ein (z. B. `delete` matcht jede Lösch-Aktion).

## Hinweise

- Der Org-Audit-Log ist **append-only**. Einträge können in der Oberfläche weder bearbeitet noch gelöscht werden.
- Login- und Passwort-Ereignisse sind aus dem org-bezogenen Log absichtlich **ausgeschlossen**, weil sie organisationsübergreifend sind. Superadmins sehen sie über die API — siehe [Globales Audit-Log untersuchen](../../operate/investigate-the-global-audit-log/).
- Für das Pro-Feld-Schema siehe [API: Audit-Logs](../../../reference/api/).
- Häufige Detektiv-Szenarien:
  - „Wer hat das Kind gelöscht?" — nach Aktion `child_delete` filtern.
  - „Was hat sich gestern geändert?" — nach Datumsbereich filtern.
  - „Hat jemand einen Bescheid bearbeitet?" — nach Aktion `government_funding_bill_*` filtern.
