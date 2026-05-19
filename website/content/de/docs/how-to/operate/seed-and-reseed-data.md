---
title: Datenbank seeden und neu seeden
weight: 4
---

Sie wollen die Demo-Organisation „Kita Sonnenschein" mit Bereichen, Mitarbeitenden, Kindern und Verträgen laden (oder neu laden). Nützlich für Entwicklung, Demos und Tests.

{{< callout type="warning" >}}
**Niemals Testdaten in eine Produktivdatenbank seeden.** Die Demo-Konten nutzen das öffentliche Standard-Passwort `supersecret`; jede geseedete Umgebung hat dieselben Zugangsdaten. Behandeln Sie sie als kompromittiert, sobald der Host erreichbar ist. Der Loader setzt das durch: `SEED_TEST_DATA=true` zusammen mit `SECURE_COOKIES=true` lässt die API den Start verweigern.
{{< /callout >}}

## Beim ersten Start per Env-Var seeden

Setzen Sie diese Env-Vars vor dem Hochfahren der API. Sie greifen nur auf einer frischen Datenbank:

```
SEED_TEST_DATA=true
SEED_ADMIN_EMAIL=superadmin@example.com
SEED_ADMIN_PASSWORD=supersecret
SEED_ADMIN_NAME=Super Admin
GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml
GOVERNMENT_FUNDING_SEED_STATE=berlin
```

## Während der Entwicklung neu seeden

```bash
make dev-fresh   # droppt die DB, führt Migrationen neu aus, seedet neu
```

Oder, wenn Sie schon einen laufenden Stack haben und neu starten wollen:

```bash
make docker-reset   # stoppt den Stack und droppt Volumes (kein Re-Seed in diesem Schritt)
make dev            # bringt ihn zurück; der Seed läuft auf der leeren DB
```

`make dev-fresh` ist das Ein-Schritt-Äquivalent.

## Was geseedet wird

- Organisation **Kita Sonnenschein** mit drei Bereichen (Nest, Nestflüchter, Große).
- ~120 Kinder mit realistischen Altersverteilungen und aktiven Betreuungsverträgen.
- ~35 Mitarbeitende mit Arbeitsverträgen über alle Bereiche verteilt.
- TVöD-SuE-2024- und Minijob-Entgelttabellen.
- Berliner Fördersätze aus dem YAML.
- Drei Demo-Nutzer:innen — `superadmin@example.com`, `admin@example.com`, `manager@example.com`, alle mit Passwort `supersecret` — für die Rollen Superadmin, Admin und Manager. Vor jeder Veröffentlichung ändern oder entfernen.

## Hinweise

- Der Seed füllt eine *neue* Datenbank. Er merged nicht mit bestehenden Daten; auf einer befüllten DB ist es ein No-Op für bestehende Zeilen.
- Die Daten sind fiktiv. Keiner der Namen, Gutscheinnummern oder Adressen entspricht echten Personen oder Kitas.
- Für Testdaten siehe auch die Integrationstest-Fixtures in `internal/testutil/`.
