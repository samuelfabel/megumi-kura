# Autenticação (Megumi Kura)

## Resumo

A área **pública** não exige login.

A área **admin** exige autenticação. O fluxo atual é:

1. `POST /api/v1/auth/login` com `username` + `password`
2. O servidor valida a senha com **bcrypt**
3. Emite um **token assinado** e o envia como cookie HttpOnly `mk_session`
4. Rotas `/api/v1/admin/*` e `GET /api/v1/auth/me` exigem esse token (cookie ou `Authorization: Bearer`)

## Não é JWT

Apesar de parecer “token com assinatura”, **não usamos JWT** (RFC 7519).

| | Este projeto | JWT típico |
| --- | --- | --- |
| Formato | `base64url(payload) . base64url(hmac)` | `header.payload.signature` |
| Payload | texto `userID\|unixExpiry` | JSON de claims |
| Header | não existe | `{"alg":"HS256",...}` |
| Algoritmo | **HMAC-SHA256** com `MK_SESSION_SECRET` | frequentemente HS256/RS256 |

É um **cookie de sessão assinado** (padrão próximo a signed cookies), não uma biblioteca JWT.

## Como o token é montado

```text
payload   = "{userID}|{expUnix}"
signature = HMAC-SHA256(payload, MK_SESSION_SECRET)  // em base64url
token     = base64url(payload) + "." + signature
```

Validação no middleware:

1. Lê cookie `mk_session` **ou** header `Authorization: Bearer …`
2. Recalcula o HMAC e compara em tempo constante
3. Rejeita se expirado ou assinatura inválida
4. Carrega o usuário no banco e exige `active = true`

## Transporte

- **Cookie HttpOnly** (`mk_session`): caminho padrão do browser no mesmo origin
- **Bearer** (opcional): útil para clientes HTTP / scripts, mesmo formato de token

`POST /api/v1/auth/logout` apenas limpa o cookie. Não há denylist no servidor; a sessão deixa de ser enviada pelo cliente e o token eventualmente expira.

## Senhas

- Hash **bcrypt** na tabela `users.password_hash`
- Nunca logar senha em texto puro
- Seed local: `admin` / `admin123` (somente desenvolvimento)

## Configuração

| Variável | Função |
| --- | --- |
| `MK_SESSION_SECRET` | chave HMAC (obrigatório mudar fora do local) |
| `MK_SESSION_HOURS` | validade do token (padrão 24) |
| `MK_COOKIE_SECURE` | `Secure` no cookie (use `true` em HTTPS) |

## Evolução possível

Se no futuro quisermos JWT de verdade (claims JSON, `kid`, refresh tokens), isso seria uma mudança explícita — o formato atual é deliberadamente pequeno e sem dependência extra.
