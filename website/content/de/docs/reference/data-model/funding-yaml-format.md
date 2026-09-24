---
title: Förder-YAML-Format
weight: 1
---

Das Förder-YAML-Format beschreibt die Kita-Fördersätze eines Bundeslandes. Die Berliner Referenz wird unter [`configs/government-fundings/berlin.yaml`](https://github.com/eenemeene/kitamanager-go/blob/main/configs/government-fundings/berlin.yaml) ausgeliefert. Dasselbe Format wird von `POST /api/v1/government-funding-rates/import` und vom Env-Var `GOVERNMENT_FUNDING_SEED_PATH` beim ersten Start gelesen.

Für *wie* die Sätze aktualisiert werden, siehe das Operator-How-To [Berliner Fördersätze aktualisieren](../../../how-to/operate/update-government-funding-rates/).

## Top-Level-Form

Eine YAML-Datei ist eine Liste von **Förder-Konfigurationen**. Jede Konfiguration repräsentiert eine zeitlich begrenzte Satz-Tabelle und enthält `entries` (in der API auch Zeiträume genannt).

```yaml
- to: ''                       # leer = offen
  from: '2026-08-01'           # ISO-Datum
  full_time_weekly_hours: 39   # was als Vollzeit zählt
  required_keys:               # Eigenschafts-Keys, die jeder Vertrag tragen muss
    - care_type
  comment: 'Anlage 1a Nr. XXXIX - Aufschlag für Praxisunterstützungssystem (45€ pro Kind/Jahr) inklusive'
  entries:
    - age: [0, 8]              # min/max Alter in Jahren
      properties: [...]
    - age: [0, 1]
      properties: [...]
    # ... weitere Altersbänder
```

## Entry-Eigenschaften

Jeder Eintrag innerhalb einer Konfiguration ist eine Liste von **Eigenschaften**-Sätzen, die für Kinder im Altersbereich des Eintrags gelten. Eine Eigenschaft hat `key`, `value`, `label`, `payment` (in **EUR als Dezimalzahl**) und `requirement` (VZÄ-Personalbedarf pro Kind für diese Eigenschaft).

```yaml
- key: care_type
  value: ganztag
  label: Full-Time (up to 9h)
  payment: 2494.91         # EUR; der Importer konvertiert intern zu Cents
  requirement: 0.355       # 0.355 VZÄ-Personalstunden pro Kind
```

### Betreuungsarten

| `key` | `value` | Bedeutung |
|---|---|---|
| `care_type` | `ganztag erweitert` | Ganztag erweitert (>9h) |
| `care_type` | `ganztag` | Ganztag (≤9h) |
| `care_type` | `teilzeit` | Teilzeit (≤7h) |
| `care_type` | `halbtag` | Halbtag (≤5h) |

### Zuschläge

| `key` | `value` | `label` | Bedeutung |
|---|---|---|---|
| `ndh` | `ndh` | NdH | nichtdeutsche Herkunftssprache — siehe [Wie die Förderung in Berlin funktioniert](../../../explanation/how-funding-works-in-berlin/) |
| `qm/mss` | `qm/mss` | QM/MSS | Quartiersmanagement / Monitoring Soziale Stadtentwicklung |
| `integration` | `integration a` | Integration A | Integrationsstatus A unter Eingliederungshilfe |
| `integration` | `integration b` | Integration B | Integrationsstatus B (höherer Förderbedarf) |

### Universelle Abzüge

`apply_to_all_contracts: true` lässt eine Eigenschaft auf jeden Vertrag anwenden, unabhängig von anderen Eigenschaften. Wird für den Eltern-Essensbeitrag genutzt:

```yaml
- key: parent
  value: meals
  label: Parent Value Meal
  payment: -23.0           # EUR (Abzug)
  requirement: 0
  apply_to_all_contracts: true
```

## Geld in YAML vs. Geld innerhalb von KitaManager

Das Förder-YAML nutzt **dezimalen EUR** für `payment`, damit eine handgepflegte Datei lesbar ist. Beim Import (`POST /api/v1/government-funding-rates/import` und der Startup-Loader für `GOVERNMENT_FUNDING_SEED_PATH`) wird der Wert in ganzzahlige Cents konvertiert und so gespeichert — `int(math.Round(eur * 100))`. Jede interne Berechnung, jede API-Antwort und jede Datenbankspalte ist in **Cents**. Round-Trip-Export zurück nach YAML emittiert wieder dezimalen EUR.

Für die Speicherform (Cents) und die Floating-Point-Falle, die sie motiviert, siehe [Warum Geldbeträge als Cents gespeichert werden](../../../explanation/why-money-is-stored-as-cents/).

## Pflicht-Eigenschaften (`required_keys`)

`required_keys` nennt die Vertrags-Eigenschafts-Keys, die ein Kindervertrag in
diesem Zeitraum tragen muss. Fehlt einer, wird der Vertrag beim Anlegen,
Ändern, Korrigieren und beim Import abgelehnt.

Die Liste steht **am Zeitraum**, nicht an der einzelnen Eigenschaft. Ob ein Key
Pflicht ist, ist eine Eigenschaft des *Keys*: Berlin verteilt `care_type` über
4 Werte × 3 Altersbänder × 15 Zeiträume, und 180 einzeln setzbare Flags könnten
sich widersprechen. Am Zeitraum gilt die Regel zudem automatisch je Bundesland
und je Datum — ein Zeitraum, der ab 2027 einen weiteren Key verlangt, ändert die
Prüfung ohne Codeänderung.

Ohne `required_keys` verlangt ein Zeitraum nichts; das ist das Verhalten aller
Zeiträume vor Einführung des Feldes.

Geprüft wird zusätzlich, dass jeder gesetzte Key/Wert in diesem Zeitraum
deklariert ist und genau **einen** Wert trägt. Ein Tippfehler, eine abweichende
Groß-/Kleinschreibung oder eine Liste zweier Betreuungsarten wird abgelehnt —
zuvor wurden sie angenommen und führten zu einem stillschweigend falschen
Förderbetrag.

Das **Alter** wird hier bewusst *nicht* geprüft: eine für ein dreijähriges Kind
gültige Betreuungsart ist es mit neun nicht mehr, und das Kind altert während
der Vertragslaufzeit. Geprüft wird die Existenz im Zeitraum.
