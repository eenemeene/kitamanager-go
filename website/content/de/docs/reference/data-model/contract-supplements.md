---
title: Vertrags-Zuschläge
weight: 2
---

Die Zuschläge, die Sie an einen Betreuungsvertrag eines Kindes anhängen können. Jeder ist ein `key`/`value`-Paar im Properties-JSON des Vertrags. Jeder bildet eine Zeile im Förder-YAML mit einem Zahlbetrag und einem Personalbedarf ab.

Für menschliche Definitionen siehe das [Glossar](../../glossary/). Für die Einbettung in die Förder-Berechnung siehe [Wie Vertragseigenschaften die Förderung bestimmen](../../../explanation/how-contract-properties-determine-funding/).

## Zuschläge (Berlin)

| UI-Beschriftung | Vertrags-`key` | Vertrags-`value` | YAML-Ort | Was es hinzufügt |
|---|---|---|---|---|
| NdH | `ndh` | `ndh` | pro-Altersband-Eintrag | kleiner pro-Kind-Förderzuschlag + zusätzliches VZÄ |
| QM/MSS | `qm/mss` | `qm/mss` | pro-Altersband-Eintrag | pro-Kind-Zuschlag (nur wenn Kita in QM/MSS-Gebiet) |
| Integration A | `integration` | `integration a` | pro-Altersband-Eintrag | größerer pro-Kind-Zuschlag + deutlich höherer VZÄ-Bedarf |
| Integration B | `integration` | `integration b` | pro-Altersband-Eintrag | noch größerer Zuschlag + höheres VZÄ |

## Betreuungsarten (auch eine Vertragseigenschaft, kein Zuschlag)

| UI-Beschriftung | `key` | `value` |
|---|---|---|
| Halbtag (≤5h) | `care_type` | `halbtag` |
| Teilzeit (≤7h) | `care_type` | `teilzeit` |
| Ganztag (≤9h) | `care_type` | `ganztag` |
| Ganztag erweitert (>9h) | `care_type` | `ganztag erweitert` |

## Universelle Abzüge

| UI-Beschriftung | `key` | `value` | Hinweise |
|---|---|---|---|
| Eltern-Essensbeitrag | `parent` | `meals` | gilt für jeden Betreuungsvertrag; derzeit −€23/Monat |

## Werte aus dem Förder-YAML lesen

Die exakten `payment`- und `requirement`-Werte hängen vom aktiven Zeitraum in `configs/government-fundings/berlin.yaml` ab. Siehe [Förder-YAML-Format](../funding-yaml-format/) für die Datei-Form.
