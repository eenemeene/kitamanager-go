---
title: Wie „heute“ und die Zeitzone funktionieren
weight: 6
---

KitaManager trifft viele „ist dieser Vertrag heute aktiv?“-Entscheidungen: welche Kinder diese Woche im Anwesenheits-Raster erscheinen, welche Verträge das Personalstunden-Diagramm für den aktuellen Monat zählt, ob ein Stufenaufstieg „jetzt fällig“ ist. Die Antwort auf „was ist heute?“ muss zur Wand-Uhr der Nutzer:in passen — nicht zur UTC-Uhr des Servers.

## Die Regel

Jede „heutige Kalenderdatum“-Entscheidung läuft durch `models.Today()`. Es liefert die UTC-Mitternacht des aktuellen Kalenderdatums in der **Anwendungs-**Zeitzone — `Europe/Berlin` als Default, übersteuerbar über die Env-Var `KITAMANAGER_TIMEZONE`.

Warum das wichtig ist: Wenn Sie um 23:30 Berliner Ortszeit am 30. September fragen „ist dieser Vertrag heute aktiv?“, denkt die UTC-Uhr des Servers, es sei 22:30 am 30. September — gleiche Antwort. Aber um 01:00 Berliner Ortszeit am 1. Oktober sagt die UTC-Uhr 23:00 am 30. September — der Vortag. Eine Berliner Nutzer:in würde „heute ist 1. Oktober“ erwarten; eine naive UTC-Trunkierung sagt 30. September, und ein Vertrag mit Beginn 1. Oktober erschiene jede Nacht eine Stunde lang als inaktiv.

## Wo es auftaucht

- Das Anwesenheits-Raster listet Kinder, deren Verträge „heute“ in Berlin aktiv sind.
- Die KPI-Kacheln des Dashboards nutzen den aktuellen Berliner Monat.
- Das Stufenaufstiegs-Widget nutzt Berliner „heute“, um zu entscheiden, welche Stufen fällig sind.
- Das auto-abgeleitete Anwesenheitsdatum beim Antippen einer Zelle ist Berliner „heute“.
- Der Future-Birthdate-Guard („Sie können kein in der Zukunft geborenes Kind anlegen“) nutzt Berliner „heute“.

`time.Now()` ist weiterhin der richtige Aufruf für *Instant*-Zeitstempel — Audit-Log-Einträge, JWT-Issued-At, MFA-Ablauf, Anwesenheits-Check-in/-out — weil diese den präzisen Moment wollen, nicht ein Kalenderdatum.

## Zeitzone ändern

`KITAMANAGER_TIMEZONE=Europe/Vienna` (oder einen beliebigen IANA-Zonennamen) setzen und die API neu starten. Der Container liefert eingebettete tzdata, sodass jede Zone unabhängig vom Basis-Image auflöst.

Wenn Sie die Zeitzone auf einem laufenden System mit bestehenden Daten ändern, verschieben sich nur *zukünftige* „heute“-Berechnungen. Bereits erfasste Anwesenheitsdaten und Audit-Zeitstempel bleiben unverändert — sie waren das richtige Kalenderdatum für die vorherige Zeitzone.

## „Heute“ in Tests pinnen

Für Entwickler:innen: `models.SetNow(instant)` übersteuert die Zeitquelle für die Dauer eines Tests. Die Schnittstelle existiert, damit Datumswechsel-Bugs in CI reproduzierbar sind — ohne sie treten sie nur auf, wenn der Runner zufällig im richtigen Moment die Zeitzonen-Grenze kreuzt.
