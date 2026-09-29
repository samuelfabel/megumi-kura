# Agent notes — Megumi Kura

## Imported skills

Skills from [open-circle/agent-skills](https://github.com/open-circle/agent-skills)
are vendored under `.agents/skills/`:

- `valibot` — schema validation with Valibot
- `formisch` — form handling with Formisch

Use them when adding frontend validation or forms.

## Source of truth

Application code lives in this repository (`megumi-kura`). Planning may happen elsewhere; do not treat a private spec repo as the place to ship product code.

## Public project rules

- Branch names follow conventional prefixes (`feat/`, `fix/`, `docs/`, `chore/`). **Never** use a `cursor/` prefix.
- Never add `Co-authored-by` (or equivalent) trailers to commits.
- Do not put ADNIPO branding as a core dependency.
- Project name is always ASCII: `Megumi Kura`.
- Public promise registration (`POST /api/v1/public/promises`) is for donors; names stay off the public dashboard.
- Auth is a signed session cookie (HMAC-SHA256), **not** JWT — see `docs/en/authentication.md`.
