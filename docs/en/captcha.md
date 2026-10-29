# Captcha on public promises

Public promise creation requires captcha. The provider is **configurable** via
environment variables.

## Providers

| `MK_CAPTCHA_PROVIDER` | Description |
| --- | --- |
| `math` (default) | Self-hosted arithmetic challenge (fine for local development) |
| `google` | Google reCAPTCHA v2 (“I’m not a robot”) |
| `cloudflare` | Cloudflare Turnstile |

Use `google` or `cloudflare` in production. The `math` mode is easy to automate.

## Configuration

```bash
# Google reCAPTCHA v2
MK_CAPTCHA_PROVIDER=google
MK_CAPTCHA_SITE_KEY=your_site_key
MK_CAPTCHA_SECRET_KEY=your_secret_key

# Cloudflare Turnstile
MK_CAPTCHA_PROVIDER=cloudflare
MK_CAPTCHA_SITE_KEY=your_site_key
MK_CAPTCHA_SECRET_KEY=your_secret_key

# Local dev (default)
MK_CAPTCHA_PROVIDER=math
# MK_CAPTCHA_TTL_SECONDS=600
```

- `MK_CAPTCHA_SITE_KEY` — public key (browser)
- `MK_CAPTCHA_SECRET_KEY` — server secret (`math` falls back to `MK_SESSION_SECRET`)

## Flow

1. `GET /api/v1/public/captcha` returns the active provider challenge
2. The public form renders the matching widget
3. `POST /api/v1/public/promises` (or `/promise-batches`) sends `captcha_token`
   (`captcha_answer` only for `math`)
4. Server verifies via HMAC (`math`) or the provider’s siteverify API

`GET /api/v1/public/settings` also exposes `captcha_provider` and `captcha_site_key`
(never the secret).

Admin promises do **not** require captcha.

## Offline

While offline, a promise may be queued **without** captcha. When back online,
the user solves the widget/challenge and flushes the queue (one item per check).
