---
title: Seed and re-seed the database
weight: 4
---

You want to load (or reload) the demo "Kita Sonnenschein" organisation, sections, employees, children, and contracts. Useful for development, demos, and tests.

{{< callout type="warning" >}}
**Never seed test data into a production database.** The seed names are fictional but the operation also creates demo accounts with default passwords.
{{< /callout >}}

## Seed via env var on first start

Set these env vars before bringing the API up. They only fire on a fresh database:

```
SEED_TEST_DATA=true
SEED_ADMIN_EMAIL=superadmin@example.com
SEED_ADMIN_PASSWORD=supersecret
SEED_ADMIN_NAME=Super Admin
GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml
GOVERNMENT_FUNDING_SEED_STATE=berlin
```

## Re-seed during development

```bash
make dev-fresh   # drops the DB, re-runs migrations, re-seeds
```

Or, if you already have a running stack:

```bash
make docker-reset
```

## What gets seeded

- Organisation **Kita Sonnenschein** with three sections (Nest, Nestflüchter, Große).
- ~120 children with realistic age distributions and active care contracts.
- ~35 employees with employment contracts spread across all sections.
- TVöD-SuE 2024 and Minijob pay plans.
- Berlin funding rates from the YAML.
- Three demo users — `superadmin@example.com`, `admin@example.com`, `manager@example.com`, all with password `supersecret` — covering the superadmin, admin, and manager roles. Change them or remove them before exposing the system to anyone.

## Notes

- The seed populates a *new* database. It does not merge into existing data; running it on a populated DB is a no-op for existing rows.
- The data is fictional. None of the names, voucher numbers, or addresses correspond to real people or real Kitas.
- For test data, also see the integration test fixtures in `internal/testutil/`.
