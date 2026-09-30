# Megumi Kura — Início

Sistema open source para igrejas e comunidades controlarem doações de alimentos,
estoque, cestas básicas e promessas de contribuição.

## Conceitos

1. **Alimentos** — itens com unidade (`kg`, `g`, `L`, `un`)
2. **Estoque** — saldo derivado de movimentações `IN` / `OUT`
3. **Cesta básica** — composição configurável; quantidade possível é calculada
4. **Promessas** — expectativa futura; não são estoque real e expiram por data

## Subir localmente

```bash
cp .env.example .env
make db-migrate
make db-seed
make web-build
make dev
```

Admin local: `admin` / `admin123` (somente desenvolvimento).
