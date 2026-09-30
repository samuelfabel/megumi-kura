-- Megumi Kura development seed (idempotent where practical)
-- Uses fictional names only. Never store real personal data here.
--
-- Local admin (CHANGE before any shared environment):
--   username: admin
--   password: admin123
-- Password is stored as bcrypt hash only.

BEGIN;

-- Foods
INSERT INTO foods (code, name, unit, sort_order, active)
VALUES
    ('arroz', 'Arroz', 'kg', 1, TRUE),
    ('feijao', 'Feijão', 'kg', 2, TRUE),
    ('acucar', 'Açúcar', 'kg', 3, TRUE),
    ('oleo', 'Óleo', 'un', 4, TRUE),
    ('macarrao', 'Macarrão', 'un', 5, TRUE),
    ('farinha', 'Farinha de trigo', 'kg', 6, TRUE),
    ('cafe', 'Café', 'kg', 7, TRUE),
    ('leite', 'Leite', 'un', 8, TRUE),
    ('sal', 'Sal', 'kg', 9, TRUE)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    unit = EXCLUDED.unit,
    sort_order = EXCLUDED.sort_order,
    active = EXCLUDED.active,
    updated_at = NOW();

-- Basket composition (replace seed basket)
DELETE FROM basket_items
WHERE food_id IN (SELECT id FROM foods WHERE code IN (
    'arroz','feijao','acucar','oleo','macarrao','farinha','cafe','leite','sal'
));

INSERT INTO basket_items (food_id, quantity)
SELECT f.id, v.qty::numeric
FROM (VALUES
    ('arroz', '5'),
    ('feijao', '2'),
    ('acucar', '2'),
    ('oleo', '2'),
    ('macarrao', '2'),
    ('farinha', '1'),
    ('cafe', '0.5'),
    ('leite', '2'),
    ('sal', '1')
) AS v(code, qty)
JOIN foods f ON f.code = v.code;

-- Initial stock movements (remove previous seed movements, then re-insert)
DELETE FROM stock_movements WHERE note = 'seed:initial-in';

INSERT INTO stock_movements (food_id, movement_type, quantity, note)
SELECT f.id, 'IN', v.qty::numeric, 'seed:initial-in'
FROM (VALUES
    ('arroz', '62'),
    ('feijao', '38'),
    ('acucar', '42'),
    ('oleo', '28'),
    ('macarrao', '30'),
    ('farinha', '15'),
    ('cafe', '12'),
    ('leite', '24'),
    ('sal', '10')
) AS v(code, qty)
JOIN foods f ON f.code = v.code;

-- Promises for today + one expired promise
-- Wipe all promises so re-seed stays deterministic in local/dev.
DELETE FROM promises;

INSERT INTO promises (food_id, person_name, quantity, promise_date)
SELECT f.id, v.person, v.qty::numeric, CURRENT_DATE
FROM (VALUES
    ('arroz', 'Joao Silva', '15'),
    ('feijao', 'Maria Santos', '8'),
    ('acucar', 'Carlos Oliveira', '10'),
    ('oleo', 'Ana Souza', '6'),
    ('macarrao', 'Joao Silva', '8')
) AS v(code, person, qty)
JOIN foods f ON f.code = v.code;

-- Expired promise (must not appear on public dashboard)
INSERT INTO promises (food_id, person_name, quantity, promise_date)
SELECT f.id, 'Pedro Lima', 5, CURRENT_DATE - INTERVAL '2 days'
FROM foods f WHERE f.code = 'arroz';

-- Admin user (bcrypt of admin123)
INSERT INTO users (username, password_hash, display_name, active)
VALUES (
    'admin',
    '$2a$10$Myfq4lXJvP7gEpVFDXTA9.INSfWf3iC4eCcubSwOHnD.3HJwUZIge',
    'Administrador',
    TRUE
)
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    display_name = EXCLUDED.display_name,
    active = TRUE,
    updated_at = NOW();

-- Settings
INSERT INTO settings (key, value) VALUES
    ('site.name', 'Megumi Kura'),
    ('site.description', 'Community Food Management'),
    ('locale.default', 'pt-BR'),
    ('theme.primary', '#2E7D32'),
    ('theme.secondary', '#795548'),
    ('theme.accent', '#F9A825'),
    ('need.arroz', '100'),
    ('need.feijao', '40'),
    ('need.acucar', '40'),
    ('need.oleo', '40'),
    ('need.macarrao', '40'),
    ('need.farinha', '20'),
    ('need.cafe', '15'),
    ('need.leite', '40'),
    ('need.sal', '15')
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = NOW();

-- Item categories (admin-defined classifications)
INSERT INTO item_categories (code, name, sort_order) VALUES
    ('alimentos', 'Alimentos', 1),
    ('higiene', 'Higiene pessoal', 2),
    ('geriatria', 'Cuidados geriatricos', 3),
    ('limpeza', 'Limpeza', 4)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

UPDATE foods
SET category_id = (SELECT id FROM item_categories WHERE code = 'alimentos')
WHERE category_id IS NULL
   OR category_id = (SELECT id FROM item_categories WHERE code = 'alimentos');

-- Non-food items
INSERT INTO foods (code, name, unit, sort_order, active, category_id)
SELECT v.code, v.name, v.unit, v.sort_order, TRUE, c.id
FROM (VALUES
    ('fralda_g', 'Fralda geriatrica G', 'un', 20, 'geriatria'),
    ('fralda_m', 'Fralda geriatrica M', 'un', 21, 'geriatria'),
    ('sabonete', 'Sabonete', 'un', 30, 'higiene'),
    ('shampoo', 'Shampoo', 'un', 31, 'higiene'),
    ('pasta_dente', 'Pasta de dente', 'un', 32, 'higiene'),
    ('papel_hig', 'Papel higienico', 'un', 33, 'higiene'),
    ('detergente', 'Detergente', 'un', 40, 'limpeza')
) AS v(code, name, unit, sort_order, cat_code)
JOIN item_categories c ON c.code = v.cat_code
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    unit = EXCLUDED.unit,
    sort_order = EXCLUDED.sort_order,
    category_id = EXCLUDED.category_id,
    active = TRUE,
    updated_at = NOW();

DELETE FROM stock_movements WHERE note = 'seed:nonfood-in';
INSERT INTO stock_movements (food_id, movement_type, quantity, note)
SELECT f.id, 'IN', v.qty::numeric, 'seed:nonfood-in'
FROM (VALUES
    ('fralda_g', '40'),
    ('fralda_m', '35'),
    ('sabonete', '50'),
    ('shampoo', '20'),
    ('pasta_dente', '25'),
    ('papel_hig', '60'),
    ('detergente', '18')
) AS v(code, qty)
JOIN foods f ON f.code = v.code;

-- Community needs (sporadic + recurring)
DELETE FROM community_needs;
INSERT INTO community_needs (food_id, title, quantity, need_kind, frequency, active, starts_on)
SELECT f.id, v.title, v.qty::numeric, v.kind, v.freq, TRUE, CURRENT_DATE
FROM (VALUES
    ('fralda_g', 'Campanha fraldas G', '80', 'sporadic', NULL),
    ('sabonete', 'Repos mensal higiene', '100', 'recurring', 'monthly'),
    ('arroz', 'Reforco mensal de arroz', '100', 'recurring', 'monthly')
) AS v(code, title, qty, kind, freq)
JOIN foods f ON f.code = v.code;

-- People (fictional beneficiaries)
DELETE FROM delivery_items;
DELETE FROM deliveries;
DELETE FROM person_needs;
DELETE FROM people;

INSERT INTO people (full_name, notes, active) VALUES
    ('Helena Costa', 'Familia com 3 pessoas', TRUE),
    ('Roberto Alves', 'Necessita fraldas periodicamente', TRUE),
    ('Lucia Mendes', 'Prioridade higiene', TRUE);

INSERT INTO person_needs (person_id, food_id, quantity, need_kind, frequency, notes)
SELECT p.id, f.id, v.qty::numeric, v.kind, v.freq, v.notes
FROM (VALUES
    ('Helena Costa', 'arroz', '5', 'recurring', 'monthly', 'Cesta mensal'),
    ('Helena Costa', 'feijao', '2', 'recurring', 'monthly', 'Cesta mensal'),
    ('Helena Costa', 'sabonete', '2', 'recurring', 'monthly', ''),
    ('Roberto Alves', 'fralda_g', '20', 'recurring', 'biweekly', 'Uso continuo'),
    ('Roberto Alves', 'papel_hig', '4', 'recurring', 'monthly', ''),
    ('Lucia Mendes', 'shampoo', '2', 'sporadic', NULL, 'Pedido pontual')
) AS v(person, code, qty, kind, freq, notes)
JOIN people p ON p.full_name = v.person
JOIN foods f ON f.code = v.code;

-- Sample delivery: Helena received a basket with these products
WITH d AS (
    INSERT INTO deliveries (person_id, delivered_on, note, deduct_stock, created_by)
    SELECT p.id, CURRENT_DATE, 'seed:entrega-cesta', FALSE, u.id
    FROM people p
    CROSS JOIN users u
    WHERE p.full_name = 'Helena Costa' AND u.username = 'admin'
    RETURNING id
)
INSERT INTO delivery_items (delivery_id, food_id, quantity)
SELECT d.id, f.id, v.qty::numeric
FROM d
CROSS JOIN (VALUES
    ('arroz', '5'),
    ('feijao', '2'),
    ('oleo', '2'),
    ('sabonete', '2')
) AS v(code, qty)
JOIN foods f ON f.code = v.code;

COMMIT;
