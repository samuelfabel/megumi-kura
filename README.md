# Megumi Kura

Open-source system for churches and communities to manage food donations, stock,
basic-basket needs, and contribution promises.

> Project name is always written in ASCII: **Megumi Kura**

## Features (foundation)

- Public dashboard with current stock, complete baskets, and today's promises
- **Public promise registration** for donors (no login; queues offline and syncs when online)
- Stock based on `IN` / `OUT` movements (not a mutable balance column)
- Promises as future expectations (not real stock), filtered by date
- Admin authentication for stock, foods, basket composition, and promises
- Single Go binary + PostgreSQL; frontend compiled to static files

## Stack

| Layer | Choice |
| --- | --- |
| Backend | Go, Gin, `database/sql`, PostgreSQL |
| Frontend | React, TypeScript, Vite, i18n |
| Production | Go serves `web/dist` — no Node.js required at runtime |

## Quick start

```bash
cp .env.example .env
# Start PostgreSQL, then:
make db-migrate
make db-seed
make web-build
make dev
```

Open `http://localhost:8080`.

### Development admin (local only)

| Field | Value |
| --- | --- |
| username | `admin` |
| password | `admin123` |

Change before any shared deployment.

## Make targets

| Target | Description |
| --- | --- |
| `make dev` | Run API (serves API + static UI) |
| `make test` | Run Go tests |
| `make build` | Build Go binary and frontend |
| `make db-migrate` | Apply PostgreSQL migrations |
| `make db-seed` | Load development seed |
| `make web-build` | Compile frontend to `web/dist` |

## Documentation

- [Português (Brasil)](README.pt-BR.md)
- [docs/en](docs/en/) · [docs/pt-BR](docs/pt-BR/) · [docs/es](docs/es/) · [docs/zh](docs/zh/) · [docs/ja](docs/ja/)
- Authentication: [docs/en/authentication.md](docs/en/authentication.md) · [docs/pt-BR/autenticacao.md](docs/pt-BR/autenticacao.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Public development uses Issues + Pull Requests on this repository.

## License

[MIT](LICENSE)
