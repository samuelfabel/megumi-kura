# Megumi Kura

Sistema open source para igrejas e comunidades controlarem doações de alimentos,
estoque, necessidades de cestas básicas e promessas de contribuição.

> O nome do projeto deve ser sempre escrito em ASCII: **Megumi Kura**

## Funcionalidades (fundação)

- Dashboard público com estoque, cestas completas e promessas do dia
- **Registro público de promessa** para doadores (sem login; fila offline e envio automático)
- Estoque derivado de movimentações `IN` / `OUT`
- Promessas como expectativa futura (não entram no estoque real)
- Autenticação administrativa
- Um binário Go + PostgreSQL; frontend compilado para arquivos estáticos

## Início rápido

```bash
cp .env.example .env
make db-migrate
make db-seed
make web-build
make dev
```

Abra `http://localhost:8080`.

### Admin de desenvolvimento (somente local)

- usuário: `admin`
- senha: `admin123`

Altere antes de qualquer ambiente compartilhado.

## Documentação

Veja também [README.md](README.md) (inglês) e a pasta `docs/pt-BR/`.

## Licença

[MIT](LICENSE)
