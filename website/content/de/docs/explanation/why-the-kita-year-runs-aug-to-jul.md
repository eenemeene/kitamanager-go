---
title: Warum das Kita-Jahr von August bis Juli läuft
weight: 4
---

KitaManager behandelt das Kita-Jahr als **1. August → 31. Juli**, nicht als Kalenderjahr. Das ist die deutsche Schuljahres-Konvention: Schulen und Kitas führen Aufnahme, Personal und Haushalt über diesen Zeitraum.

Falls Sie als Entwickler:in auf Finanz-Berichte schauen und sich wundern, warum das kumulierte Saldo im August „zurückgesetzt" wird: diese Seite erklärt es.

## Was dem Kita-Jahr folgt

- **Kumuliertes Saldo** auf der Finanzübersicht wird am 1. August auf null zurückgesetzt.
- **Förder-Zeiträume** im Berliner Kostenblatt beginnen typischerweise am 1. August.
- **TVöD-SuE-Stufenaufstiege** für Erzieher:innen sind oft an die Schuljahres-Grenze ausgerichtet.
- **Bereichs-Zuordnungen** für Kinder rotieren traditionell zum Schuljahres-Wechsel (die Nest-Kohorte rückt nach Nestflüchter etc.).

## Was dem Kita-Jahr *nicht* folgt

- Das **Kalenderjahr** ist weiterhin die Einheit für Steuerberichte, Eltern-Beitragsabrechnungen und viele externe Statistiken.
- **Audit-Log**-Einträge sind gegen den Kalender (UTC-Instanten) zeitgestempelt; Filter nutzen Kalenderdaten.
- **Anwesenheit** wird pro Kalendertag erfasst.
- **Entgelttabellen** können beliebige Von-/Bis-Bereiche haben; sie müssen sich nicht am Kita-Jahr ausrichten (obwohl TVöD-SuE-Updates es oft tun).

## Wo Kita-Jahres-Grenzen in der Oberfläche erscheinen

- Das Diagramm der Finanzübersicht hat abwechselnd schattierte Bänder, die Kita-Jahre markieren. Jedes Band ist ein August-Juli-Zeitraum.
- Die Förder-Bescheid-Upload-Seite gruppiert Bescheide nach Kita-Jahr.
- Die Defizit-Markierungen des Kumulierten-Saldo-Diagramms werden an jeder August-Grenze zurückgesetzt.

## Die richtige Zeit-Aggregation wählen

Wenn Sie *Jahr-zu-Jahr* vergleichen müssen, das Kita-Jahr nutzen. Wenn Sie *steuer-relevante Zahlen* berichten müssen, das Kalenderjahr nutzen. Die Statistik-Seite lässt Sie beliebige `from`-/`to`-Daten setzen, daher sind beide Sichten möglich.
