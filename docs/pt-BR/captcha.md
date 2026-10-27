# Captcha em promessas públicas

A inclusão de promessa pelo endpoint público exige um **captcha aritmético**
self-hosted (sem provedor externo).

## Fluxo

1. O cliente chama `GET /api/v1/public/captcha`
2. A API devolve `{ captcha_token, question }` (ex.: `"7 + 4"`)
3. O doador resolve a conta e envia `POST /api/v1/public/promises` com:
   - dados da promessa (`food_id`, `person_name`, `quantity`, `promise_date`)
   - `captcha_token` e `captcha_answer`
4. O servidor valida assinatura HMAC, validade, resposta e uso único do token

Promessas criadas pela área **admin** (`POST /api/v1/admin/promises`) **não**
exigem captcha (já autenticadas).

## Formato do token

Mesmo padrão da sessão (HMAC-SHA256 com `MK_SESSION_SECRET`):

```text
payload = "{expUnixMilli}|{a}|{b}|{nonce}"
token   = base64url(payload) + "." + base64url(HMAC-SHA256(payload))
```

A resposta correta é `a + b`. Tokens usados ou expirados são rejeitados.

## Offline

Sem conexão, a promessa pode ser enfileirada no aparelho **sem** captcha.
Ao voltar online, o usuário resolve um captcha novo e envia a fila
(um item por verificação). Não há flush silencioso automático, porque a API
exige captcha em cada `POST` público.

## Configuração

| Variável | Função |
| --- | --- |
| `MK_SESSION_SECRET` | chave HMAC do token de captcha (mesma da sessão) |
| `MK_CAPTCHA_TTL_SECONDS` | validade do desafio (padrão 600) |
