# Captcha em promessas públicas

A inclusão de promessa pelo endpoint público exige captcha. O provedor é
**configurável** por variável de ambiente.

## Provedores

| `MK_CAPTCHA_PROVIDER` | Descrição |
| --- | --- |
| `math` (padrão) | Desafio aritmético self-hosted (bom para desenvolvimento local) |
| `google` | Google reCAPTCHA v2 (“Não sou um robô”) |
| `cloudflare` | Cloudflare Turnstile |

Em produção, use `google` ou `cloudflare`. O modo `math` é fácil de automatizar.

## Configuração

```bash
# Exemplo Google reCAPTCHA v2
MK_CAPTCHA_PROVIDER=google
MK_CAPTCHA_SITE_KEY=sua_site_key
MK_CAPTCHA_SECRET_KEY=sua_secret_key

# Exemplo Cloudflare Turnstile
MK_CAPTCHA_PROVIDER=cloudflare
MK_CAPTCHA_SITE_KEY=sua_site_key
MK_CAPTCHA_SECRET_KEY=sua_secret_key

# Dev local (padrão)
MK_CAPTCHA_PROVIDER=math
# MK_CAPTCHA_TTL_SECONDS=600
```

- `MK_CAPTCHA_SITE_KEY` — chave pública (exposta ao browser)
- `MK_CAPTCHA_SECRET_KEY` — chave secreta (só no servidor; para `math`, usa `MK_SESSION_SECRET`)

## Fluxo

1. `GET /api/v1/public/captcha` devolve o desafio do provedor ativo:
   - `math`: `{ provider, captcha_token, question }`
   - `google` / `cloudflare`: `{ provider, site_key }`
2. O formulário público renderiza o widget correspondente
3. `POST /api/v1/public/promises` (ou `/promise-batches`) envia `captcha_token`
   (e `captcha_answer` só no modo `math`)
4. O servidor valida via HMAC (`math`) ou siteverify do provedor

`GET /api/v1/public/settings` também inclui `captcha_provider` e `captcha_site_key`
(sem a secret).

Promessas admin **não** exigem captcha.

## Offline

Sem conexão, a promessa pode ser enfileirada **sem** captcha. Ao voltar online,
o usuário resolve o widget/desafio e envia a fila (um item por verificação).
