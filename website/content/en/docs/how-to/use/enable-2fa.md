---
title: Enable two-factor authentication
weight: 12
---

You want to add a second factor (authenticator code or security key) to your own account so a stolen password isn't enough to sign in.

## Enable TOTP (authenticator app)

1. Open the user menu → **Settings**.
2. In the **Two-factor authentication** card, click **Enable two-factor authentication**.
3. Confirm your password.
4. Scan the QR code with your authenticator app (Google Authenticator, 1Password, Authy, Bitwarden, etc.) — or enter the secret manually.
5. Enter the 6-digit code from your app and click **Enable two-factor**.
6. **Save the recovery codes shown next**. They are displayed exactly once. Print them, store them in a password manager, or write them down. Without them and without your authenticator, an admin will need to reset your account.
7. Tick the acknowledgement and click **Done**.

The next sign-in will ask for the password and then a 6-digit code.

## Add a security key (WebAuthn)

After TOTP is on, the same card has **Add security key**:

1. Click **Add security key**.
2. The browser opens its WebAuthn prompt — tap your YubiKey, plug in your phone passkey, or use the platform biometric.
3. Optionally label the key (e.g. "Yubikey-blue").

You can register multiple keys. The next sign-in lets you choose which factor to use.

## Other actions

- **Regenerate recovery codes** — invalidates the old codes and shows fresh ones (also one-time view). Confirm your password first.
- **Disable two-factor authentication** — removes every factor (TOTP and security keys). Requires your password and a current 2FA code.

## Notes

- 2FA is *strongly* recommended for any account that can edit data — admin, manager, staff. Members get less leverage from a stolen account but should still enable it.
- Lost both authenticator and recovery codes? An admin can reset your **password** but cannot today reset your **MFA factors** without deleting the user record. Don't lose both.
- For the team-wide rollout (turning 2FA into the default for everyone), see [Enable 2FA for your team](../../administer/enable-2fa-for-your-team/).
