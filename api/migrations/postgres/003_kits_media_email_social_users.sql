-- Kits multiplos, midia, promessas em lote, email, social, usuarios

-- ---------------------------------------------------------------------------
-- Media (icones / imagens ate 1MB)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS media_assets (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('food_icon', 'kit_icon', 'social_image', 'other')),
    filename TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL,
    size_bytes INT NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 1048576),
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE foods
    ADD COLUMN IF NOT EXISTS icon_media_id BIGINT REFERENCES media_assets(id) ON DELETE SET NULL;

-- ---------------------------------------------------------------------------
-- Kits (grupos dinamicos: cesta basica, kit idosos, etc.)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kits (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon_media_id BIGINT NULL REFERENCES media_assets(id) ON DELETE SET NULL,
    sort_order INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS kit_items (
    id BIGSERIAL PRIMARY KEY,
    kit_id BIGINT NOT NULL REFERENCES kits(id) ON DELETE CASCADE,
    food_id BIGINT NOT NULL REFERENCES foods(id) ON DELETE RESTRICT,
    quantity NUMERIC(14, 3) NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kit_id, food_id)
);

CREATE INDEX IF NOT EXISTS idx_kit_items_kit ON kit_items(kit_id);

-- Migrar composicao unica atual para kit padrao
INSERT INTO kits (code, name, description, sort_order)
VALUES ('cesta_basica', 'Cesta básica', 'Composição padrão herdada', 1)
ON CONFLICT (code) DO NOTHING;

INSERT INTO kit_items (kit_id, food_id, quantity)
SELECT k.id, bi.food_id, bi.quantity
FROM basket_items bi
CROSS JOIN kits k
WHERE k.code = 'cesta_basica'
ON CONFLICT (kit_id, food_id) DO UPDATE SET quantity = EXCLUDED.quantity;

ALTER TABLE deliveries
    ADD COLUMN IF NOT EXISTS kit_id BIGINT NULL REFERENCES kits(id) ON DELETE SET NULL;

-- ---------------------------------------------------------------------------
-- Beneficiarios: frequencia de retirada
-- ---------------------------------------------------------------------------
ALTER TABLE people
    ADD COLUMN IF NOT EXISTS pickup_frequency TEXT
        CHECK (pickup_frequency IS NULL OR pickup_frequency IN ('weekly', 'biweekly', 'monthly'));

-- ---------------------------------------------------------------------------
-- Promessas em lote (lista de itens e/ou um kit)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS promise_batches (
    id BIGSERIAL PRIMARY KEY,
    person_name TEXT NOT NULL,
    kit_id BIGINT NULL REFERENCES kits(id) ON DELETE SET NULL,
    promise_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE promises
    ADD COLUMN IF NOT EXISTS batch_id BIGINT NULL REFERENCES promise_batches(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_promises_batch ON promises(batch_id);

-- ---------------------------------------------------------------------------
-- Usuarios: e-mail, bootstrap admin, permissoes, reset
-- ---------------------------------------------------------------------------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email TEXT,
    ADD COLUMN IF NOT EXISTS is_bootstrap_admin BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users SET is_bootstrap_admin = TRUE
WHERE username = 'admin' AND NOT is_bootstrap_admin;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique
    ON users (LOWER(email)) WHERE email IS NOT NULL AND email <> '';

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL CHECK (permission IN (
        'stock', 'kits', 'people', 'deliveries', 'promises',
        'social', 'email', 'users', 'settings', 'categories', 'needs'
    )),
    PRIMARY KEY (user_id, permission)
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_user ON password_reset_tokens(user_id);

-- ---------------------------------------------------------------------------
-- E-mail: inscritos, templates, dominios confiaveis, config SMTP
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS email_subscribers (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    email_normalized TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    unsubscribe_token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inactivated_reason TEXT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_email_subscribers_norm
    ON email_subscribers (email_normalized);

-- Admin NAO lista esta tabela via API.

CREATE TABLE IF NOT EXISTS trusted_email_domains (
    domain TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO trusted_email_domains (domain) VALUES
    ('gmail.com'),
    ('googlemail.com'),
    ('outlook.com'),
    ('hotmail.com'),
    ('live.com'),
    ('yahoo.com'),
    ('icloud.com'),
    ('proton.me'),
    ('protonmail.com')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS email_templates (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_html TEXT NOT NULL,
    required_tags TEXT[] NOT NULL DEFAULT '{}',
    optional_tags TEXT[] NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Templates so via migration (admin edita conteudo, nao cria/remove chaves)
INSERT INTO email_templates (key, name, subject, body_html, required_tags, optional_tags) VALUES
(
    'shortage_alert',
    'Alerta de faltas',
    '{{site_name}}: produtos em falta',
    '<p>Olá,</p><p>Estes itens estão em falta ou abaixo do limiar em <strong>{{site_name}}</strong>:</p>{{shortage_list}}<p><a href="{{unsubscribe_url}}">Cancelar inscrição</a></p>',
    ARRAY['site_name', 'shortage_list', 'unsubscribe_url'],
    ARRAY['as_of']
),
(
    'password_reset',
    'Recuperação de senha',
    '{{site_name}}: redefinir senha',
    '<p>Olá {{display_name}},</p><p>Use o link para redefinir sua senha (válido por tempo limitado):</p><p><a href="{{reset_url}}">{{reset_url}}</a></p><p>Se você não pediu isso, ignore este e-mail.</p>',
    ARRAY['site_name', 'display_name', 'reset_url'],
    ARRAY[]::TEXT[]
)
ON CONFLICT (key) DO NOTHING;

-- Limiares por alimento (abaixo disso dispara alerta); settings SMTP em settings key/value
-- settings keys: smtp.host, smtp.port, smtp.user, smtp.password, smtp.from,
-- mail.auto_enabled, mail.auto_frequency (hourly|daily|weekly), mail.global_threshold

CREATE TABLE IF NOT EXISTS food_alert_thresholds (
    food_id BIGINT PRIMARY KEY REFERENCES foods(id) ON DELETE CASCADE,
    threshold NUMERIC(14, 3) NOT NULL CHECK (threshold >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mail_send_log (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    triggered_by TEXT NOT NULL, -- auto|manual
    recipient_count INT NOT NULL DEFAULT 0,
    shortage_count INT NOT NULL DEFAULT 0,
    skipped_reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Area social
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS social_posts (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('article', 'testimonial')),
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    image_media_id BIGINT NULL REFERENCES media_assets(id) ON DELETE SET NULL,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_social_posts_published ON social_posts(published, created_at DESC);

CREATE TABLE IF NOT EXISTS social_comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    author_name TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    moderated_at TIMESTAMPTZ NULL,
    moderated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_social_comments_post ON social_comments(post_id, status);
