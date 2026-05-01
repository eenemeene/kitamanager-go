---
title: Reset a user's password
weight: 4
---

A user lost their password and can't sign in. As an admin you can issue them a new one.

## Steps

1. Open **Settings** → **Users** → click the user.
2. Click **Reset password**.
3. Enter a new initial password (the user changes it on next sign-in).
4. Click **Save**.

The user is signed out of every device. Communicate the temporary password to them through a secure channel.

## Notes

- Resetting the password does **not** disable 2FA. If the user lost both authenticator and recovery codes, password reset alone won't get them back in. Today the recovery path is to delete and re-create the user account; admin reset of MFA factors is on the roadmap.
- The reset is recorded in the audit log.
- For the *self-service* path (a user changing their own password), see [Change your password](../../use/change-password/).
