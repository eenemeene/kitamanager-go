---
title: Operate the system
weight: 3
---

Recipes for the person running the KitaManager server: deploy, back up, rotate keys, publish releases, keep funding-rate tables current. Most of these are `superadmin` or shell-level operations on the host.

For developer tasks (extending the codebase), see [Develop on the codebase](../develop/).

## Deploy

- [Deploy with Docker Compose](deploy-with-docker-compose/)
- [Back up and restore the database](back-up-and-restore/)

## Funding configuration

- [Update government funding rates](update-government-funding-rates/)
- [Seed and re-seed the database](seed-and-reseed-data/)

## Security

- [Rotate the TOTP encryption key](rotate-totp-encryption-key/)
- [Investigate the global audit log](investigate-the-global-audit-log/)

## Releases

- [Publish a release](publish-a-release/)
