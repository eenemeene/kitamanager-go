---
title: Was die Personalkennzahl bedeutet
weight: 3
---

KitaManagers Zahl zur **Personalabdeckung** ist eine ein-Zeilen-Zusammenfassung von „habe ich genug Personal für die Kinder, die ich habe?". Diese Seite erklärt, was tatsächlich berechnet wird.

## Die zwei Seiten der Gleichung

Für jeden Monat berechnet KitaManager zwei Zahlen:

- **Benötigte Stunden** — die Personalstunden, die die Förder-Konfiguration für Ihre Kinder vorsieht. Jede Vertragseigenschaft hat ein `requirement`-Feld (in VZÄ) — der pro-Kind-Personalbedarf für diese Eigenschaft. Die landesspezifischen Bedarfssummen werden für alle aktiven Kinder im Monat aufsummiert.
- **Verfügbare Stunden** — die Personalstunden, die Ihre aktiven Arbeitsverträge bereitstellen. Summe von `weekly_hours` über alle Mitarbeitenden, deren Vertrag im Monat aktiv ist, skaliert auf Monatsmenge.

Die Prozentzahl **Personalabdeckung** auf dem Dashboard ist `(verfügbar − benötigt) / benötigt × 100`, mit sinnvollem Vorzeichen begrenzt (negativ = unterbesetzt, positiv = Überschuss).

## Wie der Bedarf aus der Förder-Konfiguration kommt

Jede Eigenschaft im Förder-YAML trägt ein `requirement`:

```yaml
- key: care_type
  value: ganztag
  payment: 2494.91
  requirement: 0.355   # 0.355 VZÄ pro Kind für Ganztags-Betreuung
- key: ndh
  value: ndh
  payment: 93.51
  requirement: 0.017   # zusätzliches VZÄ für NdH-Kinder
```

Ein 1-jähriges Kind mit `ganztag` und NdH steuert `0.355 + 0.017 = 0.372` VZÄ zur benötigten Seite bei. Multipliziert mit den Wochenstunden, die als Vollzeit zählen (`full_time_weekly_hours: 39` für Berlin), und dann mit der Anzahl Wochen im Monat, ergibt das die benötigten Personalstunden für dieses eine Kind.

Summe über jedes aktive Kind im Monat → benötigte Stundensumme.

## Wie die Verfügbarkeit aus Verträgen kommt

Für jede Mitarbeiter:in:

1. Jeden Vertrag finden, der den Abrechnungsmonat überlappt.
2. Die `weekly_hours` des Vertrags × aktive Tage im Monat / Tage-im-Monat → der Beitrag des Vertrags.

Summe über alle Mitarbeitenden → verfügbare Stundensumme.

## Was die Prozentzahl tatsächlich aussagt

| Prozent | Bedeutung |
|---|---|
| `+0%` | Genau der Förder-Konfigurations-Anforderung entsprechend besetzt. |
| `+10%` | 10 % mehr verfügbares Personal als gefordert. Überschuss. |
| `−10%` | 10 % zu wenig. Kinder, deren Betreuung von den fehlenden Stunden abhängt, bekommen nicht die nach Konfiguration zugesagte Aufmerksamkeit. |
| `−40%` oder schlimmer | Schwere Unterbesetzung. Das vom Senat definierte Verhältnis wird verletzt. |

Ein dauerhafter Überschuss ist nicht immer gut — er bedeutet, dass Sie Personalstunden bezahlen, die von Ihren eingeschriebenen Kindern nicht gefordert sind. Ein dauerhaftes Defizit bedeutet, dass Sie nicht die Betreuung leisten können, die die Fördersätze annehmen.

## Sicht pro Bereich

Statistiken → Personalstunden hat sowohl ein organisations-weites Diagramm als auch eine pro-Bereich-Aufschlüsselung. Der `requirement` der Kinder wird über die Verträge ihres zugewiesenen Bereichs summiert; der `available` der Mitarbeiter:innen über deren zugewiesenen Bereich.

Eine balancierte Organisations-Gesamtsumme kann ein bereichs-internes Ungleichgewicht verbergen (ein Bereich unterbesetzt, ein anderer überbesetzt). Beide Sichten prüfen.

## Vorbehalte

- **Urlaub ist nicht modelliert.** Verfügbare Stunden nehmen an, dass Verträge ununterbrochen laufen. Echte Urlaubsabwesenheit wird nicht abgezogen.
- **Krankheit ist nicht modelliert.** Gleicher Grund.
- **Nicht-pädagogisches Personal (Hauswirtschaft, Verwaltung) sollte nicht zu `available` zählen** — wird aber, wenn Sie sie mit einer `staff_category` klassifizieren, die im Förder-YAML als nicht-pädagogisch geführt ist, korrekt ausgeschlossen. Die pro-Mitarbeiter-Sicht prüfen, wenn Zahlen merkwürdig wirken.
- **Das `requirement` der Förder-Konfiguration ist die Senats-Zahl, nicht Ihre.** Wenn Ihre Kita einen höheren pädagogischen Standard hat (mehr Personal pro Kind), zeigen Sie immer eine positive Abdeckung. Das ist Absicht — die Senats-Zahl ist die Untergrenze.
