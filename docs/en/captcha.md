# Captcha on public promises

Public promise creation requires a **self-hosted arithmetic captcha**
(no third-party provider).

## Flow

1. Client calls `GET /api/v1/public/captcha`
2. API returns `{ captcha_token, question }` (e.g. `"7 + 4"`)
3. Donor solves it and posts `POST /api/v1/public/promises` with:
   - promise fields (`food_id`, `person_name`, `quantity`, `promise_date`)
   - `captcha_token` and `captcha_answer`
4. Server checks HMAC signature, expiry, answer, and one-time use

Admin-created promises (`POST /api/v1/admin/promises`) do **not** require
captcha (already authenticated).

## Token format

Same pattern as the session cookie (HMAC-SHA256 with `MK_SESSION_SECRET`):

```text
payload = "{expUnixMilli}|{a}|{b}|{nonce}"
token   = base64url(payload) + "." + base64url(HMAC-SHA256(payload))
```

Correct answer is `a + b`. Used or expired tokens are rejected.

## Offline

While offline, a promise may be queued on the device **without** captcha.
When back online, the user solves a fresh captcha and flushes the queue
(one item per challenge). There is no silent auto-flush because every public
`POST` requires captcha.

## Configuration

| Variable | Purpose |
| --- | --- |
| `MK_SESSION_SECRET` | HMAC key for captcha tokens (same as session) |
| `MK_CAPTCHA_TTL_SECONDS` | challenge lifetime (default 600) |
