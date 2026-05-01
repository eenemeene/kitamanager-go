---
title: Enable 2FA for your team
weight: 9
---

You want to make 2FA the default for every account in your organisation that can edit data.

## Steps

1. **Enable 2FA on your own account first** following [Enable two-factor authentication](../../use/enable-2fa/). Save your recovery codes and verify the next sign-in prompts you for a code.
2. **Pick a team standard** for the authenticator app (Google Authenticator, 1Password, Authy, Bitwarden — one consistent choice means one set of instructions to support) and for where recovery codes live (personal password manager preferred).
3. **Enrol each teammate** in person or via screenshare. Walk them through [Enable 2FA](../../use/enable-2fa/), watch them save the recovery codes, sign them out and back in to verify the prompt fires.
4. **Add it to onboarding** so every new admin/manager enables 2FA on day one alongside the password change.
5. **Audit quarterly:** there's no "users without 2FA" report yet — script a quick check by calling `GET /api/v1/users/{userId}/factors` for each admin/manager (or `me` for the calling user) and chase anyone whose response has no enrolled factor.

## Notes

- Today the only recovery from "lost both authenticator and codes" is to delete + recreate the account. Tell users this when they enrol so they take the codes seriously.
- Don't communicate initial passwords or recovery codes by email — use in-person, video call, or Signal/Slack DM.
- For the related operator concern (`TOTP_ENCRYPTION_KEY` rotation invalidates every TOTP factor), see [Rotate the TOTP encryption key](../../operate/rotate-totp-encryption-key/).
