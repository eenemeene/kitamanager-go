---
title: Rotate the TOTP encryption key
weight: 5
---

You want to change `TOTP_ENCRYPTION_KEY` (e.g. as part of routine key rotation, or because you suspect it's leaked).

{{< callout type="warning" >}}
**Rotating the key invalidates every stored TOTP secret.** Every user with TOTP enrolled must re-enrol after the change. Plan a rollout window.
{{< /callout >}}

## Steps

1. Generate the new key:
   ```bash
   openssl rand -hex 32
   ```
2. Notify your users that 2FA will need to be re-enrolled at a specific time.
3. Update `TOTP_ENCRYPTION_KEY` in your environment (e.g. `.env`, your secrets manager).
4. Restart the API.
5. Existing TOTP factors are now undecryptable. Users still see their card, but verification fails.
6. Guide users through:
   - Sign in (password works as before).
   - Enter their **recovery code** when prompted for 2FA — every user should have one of these.
   - Once signed in, go to Settings → 2FA → **Disable**, then **Enable** again to re-enrol.

## If you need to disable a user's TOTP without their cooperation

Today there's no admin "reset MFA" UI. If a user has lost their authenticator and recovery codes after the rotation, the recovery path is:

1. Reset their password (which doesn't disable 2FA).
2. Delete the user record.
3. Recreate the user with the same email and a new password.

Direct admin reset of MFA factors is on the roadmap.

## Notes

- WebAuthn (security keys) is **not** affected by `TOTP_ENCRYPTION_KEY` rotation. Only TOTP secrets are encrypted with this key.
- For why the key matters at all, see [Architecture](../../../explanation/architecture/) and the comment block in `internal/config/config.go` around `TOTP_ENCRYPTION_KEY`.
- Key rotation is a useful drill even if you don't think the key is leaked. Add it to your annual security checklist.
