-- The catalogue the store tests read. The ids and the timestamps are fixed,
-- so a test can assert an exact page and an exact order. TestMain loads it
-- into a schema of its own, so no database ever holds it for long.
--
-- base_price is set here because nothing derives it yet: ADR-0027 takes it
-- from the product's SKUs, and no SKU table exists. The silver cuff has none,
-- which is the absent case.

INSERT INTO labels (id, name, slug) VALUES
    ('0194f100-0001-7000-8000-000000000001', 'Jewellery', 'jewellery'),
    ('0194f100-0002-7000-8000-000000000002', 'Homeware',  'homeware'),
    ('0194f100-0003-7000-8000-000000000003', 'Woodwork',  'woodwork');

INSERT INTO label_aliases (id, label_id, alias) VALUES
    ('0194f200-0001-7000-8000-000000000001',
     '0194f100-0001-7000-8000-000000000001', 'jewelry');

INSERT INTO products (id, shop_id, name, description, currency, base_price, created_at, updated_at) VALUES
    ('0194f000-0001-7000-8000-000000000001', 'shop-thistle', 'Copper ring',
     'A hammered copper band.',        'GBP', 24.0000, '2026-01-05 10:00:00+00', '2026-01-05 10:00:00+00'),
    ('0194f000-0002-7000-8000-000000000002', 'shop-thistle', 'Linen apron',
     'A cross-back apron in flax.',    'GBP', 32.5000, '2026-01-06 10:00:00+00', '2026-01-06 10:00:00+00'),
    ('0194f000-0003-7000-8000-000000000003', 'shop-bramble', 'Oak spoon',
     'A carved oak serving spoon.',    'EUR',  9.9500, '2026-01-07 10:00:00+00', '2026-01-07 10:00:00+00'),
    ('0194f000-0004-7000-8000-000000000004', 'shop-bramble', 'Beeswax candle',
     'A rolled beeswax taper.',        'EUR',  6.0000, '2026-01-08 10:00:00+00', '2026-01-08 10:00:00+00'),
    ('0194f000-0005-7000-8000-000000000005', 'shop-thistle', 'Silver cuff',
     'A wide silver cuff. No SKUs.',   'GBP',    NULL, '2026-01-09 10:00:00+00', '2026-01-09 10:00:00+00'),
    ('0194f000-0006-7000-8000-000000000006', 'shop-bramble', 'Slate coaster',
     'A slate coaster. No labels.',    'EUR', 12.0000, '2026-01-10 10:00:00+00', '2026-01-10 10:00:00+00');

INSERT INTO product_labels (product_id, label_id) VALUES
    ('0194f000-0001-7000-8000-000000000001', '0194f100-0001-7000-8000-000000000001'),
    ('0194f000-0002-7000-8000-000000000002', '0194f100-0002-7000-8000-000000000002'),
    ('0194f000-0003-7000-8000-000000000003', '0194f100-0002-7000-8000-000000000002'),
    ('0194f000-0003-7000-8000-000000000003', '0194f100-0003-7000-8000-000000000003'),
    ('0194f000-0004-7000-8000-000000000004', '0194f100-0002-7000-8000-000000000002'),
    ('0194f000-0005-7000-8000-000000000005', '0194f100-0001-7000-8000-000000000001');

-- Every product starts incomplete, because a SKU attaches only after the row
-- exists. The linen apron takes both its rows at one timestamp, so the read
-- has to break the tie on id to find the current status.
INSERT INTO product_status_history (product_id, status, reason, created_at) VALUES
    ('0194f000-0001-7000-8000-000000000001', 'incomplete', NULL, '2026-01-05 10:00:00+00'),
    ('0194f000-0001-7000-8000-000000000001', 'active',     NULL, '2026-01-05 11:00:00+00'),
    ('0194f000-0002-7000-8000-000000000002', 'incomplete', NULL, '2026-01-06 10:00:00+00'),
    ('0194f000-0002-7000-8000-000000000002', 'active',     NULL, '2026-01-06 10:00:00+00'),
    ('0194f000-0003-7000-8000-000000000003', 'incomplete', NULL, '2026-01-07 10:00:00+00'),
    ('0194f000-0003-7000-8000-000000000003', 'active',     NULL, '2026-01-07 11:00:00+00'),
    ('0194f000-0004-7000-8000-000000000004', 'incomplete', NULL, '2026-01-08 10:00:00+00'),
    ('0194f000-0004-7000-8000-000000000004', 'active',     NULL, '2026-01-08 11:00:00+00'),
    ('0194f000-0004-7000-8000-000000000004', 'hidden',     'The shop is on holiday.', '2026-01-08 12:00:00+00'),
    ('0194f000-0005-7000-8000-000000000005', 'incomplete', NULL, '2026-01-09 10:00:00+00'),
    ('0194f000-0006-7000-8000-000000000006', 'incomplete', NULL, '2026-01-10 10:00:00+00'),
    ('0194f000-0006-7000-8000-000000000006', 'active',     NULL, '2026-01-10 11:00:00+00');
