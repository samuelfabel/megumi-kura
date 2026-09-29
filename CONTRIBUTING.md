# Contributing to Megumi Kura

Thank you for helping improve Megumi Kura.

## Project name

Always write the project name in ASCII as:

**Megumi Kura**

## Principles

- Keep the project small and simple
- Prefer explicit SQL over heavy ORMs
- Do not introduce Redis, queues, or microservices for basic features
- Public API stays unauthenticated; admin API must validate everything on the backend
- Stock is derived from movements; promises are not stock

## Development setup

1. Copy `.env.example` to `.env` and adjust as needed
2. Start PostgreSQL
3. Run migrations: `make db-migrate`
4. Seed data: `make db-seed`
5. Build the frontend: `make web-build`
6. Start the API (serves the compiled frontend): `make dev`

Default local admin (seed only):

- username: `admin`
- password: `admin123`

Change this password before any non-local use.

## Tests

```bash
make test
```

## Git branches

Use conventional branch names — **do not** use a `cursor/` prefix:

| Prefix | Use |
| --- | --- |
| `feat/` | new feature |
| `fix/` | bug fix |
| `docs/` | documentation only |
| `chore/` | tooling, seed, configs |
| `refactor/` | no behavior change |

Examples: `feat/dashboard-promessa-publica`, `fix/stock-out-validation`.

Do not add `Co-authored-by` (or equivalent) trailers to commits.

## GitHub workflow (public repository)

Prefer this loop on [megumi-kura](https://github.com/samuelfabel/megumi-kura):

1. Open an **Issue** describing the problem or feature
2. Create a branch from `main` using conventional prefixes (`feat/`, `fix/`, `docs/`, `chore/`)
3. Open a **Pull Request** that references the issue (`Closes #N` or `Fixes #N`)
4. Keep the PR focused; wait for review before merging

Do **not** use a `cursor/` branch prefix.
Do **not** add `Co-authored-by` trailers to commits.

The `megumi-kura-spec` repository may hold planning notes; **application source of truth is `megumi-kura`**.

## Pull requests

- Keep changes focused
- Update docs when behavior changes
- Do not commit secrets, `.env`, or real church personal data
- Prefer one concern per PR
- Link related issues in the PR body

## Languages

User-facing UI strings must go through i18n keys (pt-BR, en, es, zh, ja).
