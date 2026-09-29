# Authentication (Megumi Kura)

## Summary

The **public** API does not require login.

The **admin** API requires authentication:

1. `POST /api/v1/auth/login` with `username` + `password`
2. Password checked with **bcrypt**
3. Server issues a **signed token** in HttpOnly cookie `mk_session`
4. `/api/v1/admin/*` and `GET /api/v1/auth/me` require that token (cookie or `Authorization: Bearer`)

## This is not JWT

The token is signed, but it is **not** a JWT (RFC 7519).

It is a compact **HMAC-SHA256 signed session cookie**:

```text
payload   = "{userID}|{expUnix}"
signature = HMAC-SHA256(payload, MK_SESSION_SECRET)  // base64url
token     = base64url(payload) + "." + signature
```

Clients may also send the same token as `Authorization: Bearer <token>`.

Passwords are stored as bcrypt hashes. Change `MK_SESSION_SECRET` outside local development.
