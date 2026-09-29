-- Categories, people, needs, and basket deliveries

CREATE TABLE IF NOT EXISTS item_categories (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Default category for existing foods (alimentos)
INSERT INTO item_categories (code, name, sort_order)
VALUES ('alimentos', 'Alimentos', 1)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE foods
    ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES item_categories(id);

UPDATE foods
SET category_id = (SELECT id FROM item_categories WHERE code = 'alimentos')
WHERE category_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_foods_category_id ON foods(category_id);

-- Community / campaign needs (beyond a single basket composition)
CREATE TABLE IF NOT EXISTS community_needs (
    id BIGSERIAL PRIMARY KEY,
    food_id BIGINT NOT NULL REFERENCES foods(id) ON DELETE RESTRICT,
    title TEXT NOT NULL DEFAULT '',
    quantity NUMERIC(14, 3) NOT NULL CHECK (quantity > 0),
    need_kind TEXT NOT NULL CHECK (need_kind IN ('sporadic', 'recurring')),
    frequency TEXT NULL CHECK (frequency IS NULL OR frequency IN ('weekly', 'biweekly', 'monthly')),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    starts_on DATE NOT NULL DEFAULT CURRENT_DATE,
    ends_on DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_needs_recurring_freq CHECK (
        (need_kind = 'sporadic' AND frequency IS NULL)
        OR (need_kind = 'recurring' AND frequency IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_community_needs_food ON community_needs(food_id);
CREATE INDEX IF NOT EXISTS idx_community_needs_active ON community_needs(active);

-- Beneficiaries (people who receive support)
CREATE TABLE IF NOT EXISTS people (
    id BIGSERIAL PRIMARY KEY,
    full_name TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_people_active ON people(active);
CREATE INDEX IF NOT EXISTS idx_people_name ON people(full_name);

-- Per-person needs with frequency
CREATE TABLE IF NOT EXISTS person_needs (
    id BIGSERIAL PRIMARY KEY,
    person_id BIGINT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    food_id BIGINT NOT NULL REFERENCES foods(id) ON DELETE RESTRICT,
    quantity NUMERIC(14, 3) NOT NULL CHECK (quantity > 0),
    need_kind TEXT NOT NULL CHECK (need_kind IN ('sporadic', 'recurring')),
    frequency TEXT NULL CHECK (frequency IS NULL OR frequency IN ('weekly', 'biweekly', 'monthly')),
    notes TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT person_needs_recurring_freq CHECK (
        (need_kind = 'sporadic' AND frequency IS NULL)
        OR (need_kind = 'recurring' AND frequency IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_person_needs_person ON person_needs(person_id);
CREATE INDEX IF NOT EXISTS idx_person_needs_food ON person_needs(food_id);

-- Basket / kit delivery to a person
CREATE TABLE IF NOT EXISTS deliveries (
    id BIGSERIAL PRIMARY KEY,
    person_id BIGINT NOT NULL REFERENCES people(id) ON DELETE RESTRICT,
    delivered_on DATE NOT NULL DEFAULT CURRENT_DATE,
    note TEXT NOT NULL DEFAULT '',
    deduct_stock BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deliveries_person ON deliveries(person_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_date ON deliveries(delivered_on DESC);

CREATE TABLE IF NOT EXISTS delivery_items (
    id BIGSERIAL PRIMARY KEY,
    delivery_id BIGINT NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    food_id BIGINT NOT NULL REFERENCES foods(id) ON DELETE RESTRICT,
    quantity NUMERIC(14, 3) NOT NULL CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_delivery_items_delivery ON delivery_items(delivery_id);
