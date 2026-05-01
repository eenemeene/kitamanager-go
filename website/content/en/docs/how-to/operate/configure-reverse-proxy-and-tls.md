---
title: Configure a reverse proxy and TLS
weight: 9
---

You're putting KitaManager behind HTTPS for production. The API + frontend speak plain HTTP on the host (default ports 8080 and 3000); a reverse proxy in front terminates TLS and forwards to them.

## What needs to be aligned

- TLS termination at the proxy.
- Forwarding rules: API → port 8080, frontend → port 3000.
- KitaManager env vars: `WEBAUTHN_RP_ID`, `WEBAUTHN_ORIGINS`, `CORS_ALLOW_ORIGINS`, `TRUSTED_PROXIES`, `SECURE_COOKIES`.

## Caddy (simplest)

```caddy
kitamanager.example.org {
    reverse_proxy localhost:3000
}

api.kitamanager.example.org {
    reverse_proxy localhost:8080
}
```

Caddy issues a Let's Encrypt cert automatically.

## nginx

```nginx
server {
    listen 443 ssl http2;
    server_name kitamanager.example.org;
    ssl_certificate     /etc/letsencrypt/live/kitamanager.example.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/kitamanager.example.org/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 443 ssl http2;
    server_name api.kitamanager.example.org;
    ssl_certificate     /etc/letsencrypt/live/api.kitamanager.example.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.kitamanager.example.org/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Update env vars and restart

```
TRUSTED_PROXIES=127.0.0.1/32          # the proxy's IP; trust only it for X-Forwarded-*
SECURE_COOKIES=true                    # session cookie marked Secure
WEBAUTHN_RP_ID=kitamanager.example.org # exact host the browser sees
WEBAUTHN_ORIGINS=https://kitamanager.example.org
CORS_ALLOW_ORIGINS=https://kitamanager.example.org
```

Then `docker compose restart api`. Verify by signing in and checking that the session cookie shows `Secure` in browser devtools.

## Notes

- `WEBAUTHN_RP_ID` must match the public host **exactly** — including no `https://`. Mismatched RP IDs make security keys fail with no useful error.
- If the frontend and API share one hostname (e.g. both at `kitamanager.example.org` with API at `/api/`), use one server block and route based on path. WebAuthn becomes simpler because there's only one origin.
- For the env-var reference, see [Environment variables](../../../reference/cli-and-config/env-vars/).
