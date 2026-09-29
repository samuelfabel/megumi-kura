# Megumi Kura — Getting started

Open-source system for churches and communities to manage food donations, stock,
basic baskets, and contribution promises.

## Core ideas

1. **Foods** — items with a unit (`kg`, `g`, `L`, `un`)
2. **Stock** — balance derived from `IN` / `OUT` movements
3. **Basket** — configurable composition; available count is computed
4. **Promises** — future expectations; not real stock; filtered by date

## Local setup

```bash
cp .env.example .env
make db-migrate
make db-seed
make web-build
make dev
```

Local admin: `admin` / `admin123` (development only).
