---
title: Reverse-Proxy und TLS konfigurieren
weight: 9
---

Sie stellen KitaManager hinter HTTPS für Produktion. API + Frontend sprechen einfaches HTTP auf dem Host (Standard-Ports 8080 und 3000); ein Reverse-Proxy davor terminiert TLS und leitet weiter.

## Was abzustimmen ist

- TLS-Terminierung am Proxy.
- Weiterleitung: API → Port 8080, Frontend → Port 3000.
- KitaManager-Env-Vars: `WEBAUTHN_RP_ID`, `WEBAUTHN_ORIGINS`, `CORS_ALLOW_ORIGINS`, `TRUSTED_PROXIES`, `SECURE_COOKIES`.

## Caddy (am einfachsten)

```caddy
kitamanager.example.org {
    reverse_proxy localhost:3000
}

api.kitamanager.example.org {
    reverse_proxy localhost:8080
}
```

Caddy stellt automatisch ein Let's-Encrypt-Zertifikat aus.

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

## Env-Vars aktualisieren und neu starten

```
TRUSTED_PROXIES=127.0.0.1/32          # IP des Proxys; nur diesem für X-Forwarded-* vertrauen
SECURE_COOKIES=true                    # Sitzungs-Cookie als Secure markieren
WEBAUTHN_RP_ID=kitamanager.example.org # exakter Host, den der Browser sieht
WEBAUTHN_ORIGINS=https://kitamanager.example.org
CORS_ALLOW_ORIGINS=https://kitamanager.example.org
```

Dann `docker compose restart api`. Verifizieren durch Anmelden und Prüfen in den Browser-DevTools, dass das Sitzungs-Cookie als `Secure` markiert ist.

## Hinweise

- `WEBAUTHN_RP_ID` muss **exakt** zum öffentlichen Host passen — kein `https://` davor. Nicht-passende RP IDs lassen Sicherheitsschlüssel ohne brauchbare Fehlermeldung scheitern.
- Wenn Frontend und API einen Hostnamen teilen (z. B. beide auf `kitamanager.example.org` mit API unter `/api/`), nutzen Sie einen Server-Block und routen pfadbasiert. WebAuthn wird einfacher, weil es nur einen Origin gibt.
- Für die Env-Var-Referenz siehe [Umgebungsvariablen](../../../reference/cli-and-config/env-vars/).
