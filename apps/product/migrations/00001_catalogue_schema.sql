-- The catalogue read path: products, their status history, and the labels
-- they carry. The SKU and shipping tables arrive with the changes that need
-- them, so nothing here references them yet.

-- +goose Up

-- money_value hands an amount and its currency to the application as one
-- column (ADR-0032). No write uses it. The currency half takes the type of
-- products.currency.
CREATE TYPE money_value AS (amount NUMERIC(15,4), currency TEXT);

CREATE TABLE products (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    shop_id     TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL,
    -- The ISO 4217 register lives in Go (ADR-0030). This check only stops
    -- direct SQL writing something that is not a code at all.
    currency    TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    -- The lowest price among the product's SKUs (ADR-0027). Nothing
    -- maintains it yet, because there are no SKUs, so whatever writes the
    -- row sets it. It is NULL while the product has none.
    base_price  NUMERIC(15,4),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX products_shop_id_idx ON products (shop_id);

-- A product's current status is the status of its most recent history row
-- (ADR-0019). id breaks a tie on created_at, so "most recent" is
-- deterministic. It is bookkeeping rather than an entity, which keeps it off
-- UUIDv7 (ADR-0024).
CREATE TABLE product_status_history (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    status     TEXT NOT NULL CHECK (
                   status IN ('incomplete', 'active', 'hidden',
                              'archived', 'reported', 'deleted')),
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX product_status_history_latest_idx
    ON product_status_history (product_id, created_at DESC, id DESC);

CREATE TABLE labels (
    id   UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE
);

-- An alias resolves to one label, so no two labels can claim the same
-- spelling.
CREATE TABLE label_aliases (
    id       UUID PRIMARY KEY DEFAULT uuidv7(),
    label_id UUID NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    alias    TEXT NOT NULL UNIQUE
);

CREATE INDEX label_aliases_label_id_idx ON label_aliases (label_id);

-- The assignment join. It keeps its natural key (ADR-0024).
CREATE TABLE product_labels (
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    label_id   UUID NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, label_id)
);

-- The primary key answers "which labels does this product carry". This index
-- answers "which products carry this label", which is the catalogue filter.
CREATE INDEX product_labels_label_id_idx ON product_labels (label_id);

-- +goose Down

DROP TABLE product_labels;
DROP TABLE label_aliases;
DROP TABLE labels;
DROP TABLE product_status_history;
DROP TABLE products;
DROP TYPE money_value;
