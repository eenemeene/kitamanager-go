---
title: Configure SMTP and WebAuthn
weight: 10
---

You want to enable email-driven features (currently optional) and security-key sign-in.

## SMTP (optional)

If `SMTP_HOST` is empty, email-related features are off. To enable, set:

```
SMTP_HOST=smtp.example.org
SMTP_PORT=587
SMTP_USER=kitamanager@example.org
SMTP_PASSWORD=<app password>
SMTP_FROM=KitaManager <noreply@example.org>
```

Restart the API. The first email-triggering action surfaces SMTP errors in the API log if any of the above is wrong; there's no startup connection check.

## WebAuthn (security keys)

Three env vars must match the public URL the browser sees:

```
WEBAUTHN_RP_ID=kitamanager.example.org              # host only, no scheme, no trailing slash
WEBAUTHN_RP_NAME=KitaManager — Kita Sonnenschein    # shown in the browser prompt
WEBAUTHN_ORIGINS=https://kitamanager.example.org    # full origin, comma-separated for multiple
```

After restarting, the **Add security key** button on the Settings page works. If it fails silently or the browser says "the security key isn't valid for this site", the RP ID doesn't match the URL.

## Notes

- Changing `WEBAUTHN_RP_ID` invalidates every previously enrolled security key (the key was bound to the old RP ID). Treat it as a one-time decision and pin it to your stable production hostname.
- Do *not* use the localhost RP ID in production. WebAuthn refuses to register against `localhost` from a non-secure context.
- For the reverse-proxy setup that makes these URLs work, see [Configure a reverse proxy and TLS](../configure-reverse-proxy-and-tls/).
