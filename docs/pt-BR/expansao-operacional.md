# Expansão operacional

Documenta os recursos adicionados além da fundação (kits, projeções, e-mail, social, usuários).

## Kits (grupos dinâmicos)

- Tabela `kits` + `kit_items` — vários tipos (cesta básica, kit idosos, …)
- Entrega com `kit_id` expande os itens e dá baixa de cada produto no estoque
- Ícones: upload em `POST /api/v1/admin/media` (máx. 1MB) e vínculo em kit/alimento

## Progresso na home

- Barra empilhada: **verde = estoque**, **amarelo = prometido**
- `shortage_risks` estima dias até faltar com base em necessidades recorrentes

## Projeções

- `GET /api/v1/public/projections?food_id=&days=` — série diária estoque × tempo (gráfico de linha)

## Promessas em lote

- `POST /api/v1/public/promise-batches` — lista de itens e/ou um kit (com captcha)
- Expande kit em linhas de `promises` ligadas a `promise_batches`

## Beneficiários

- `people.pickup_frequency` + `last_delivered_on` (derivado de entregas)

## E-mail de faltas

- Inscrição pública (sempre `ok`, sem enumerar)
- Domínios confiáveis + verificação MX
- Templates só via migration; admin edita HTML com tags obrigatórias
- SMTP e limiares em settings; envio manual/automático; sem lista de e-mails na API
- Unsubscribe por token; SMTP inválido inativa o inscrito

## Social

- Posts (artigo/depoimento) + comentários com moderação

## Usuários e permissões

- Bootstrap admin (`username`) sem e-mail; demais entram com e-mail
- Permissões por usuário; reset de senha via template `password_reset`
- Sem auto-cadastro público
