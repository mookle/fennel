# 0028 - Currency lives on the product, not the SKU

Status: Accepted (amended by ADR-0032)

## Context

A price never travels without an ISO-4217 currency code (ADR-0017), so `Sku` and `ResolvedSku` both carry `currency` on the wire.

The `sku` table carries a matching column but nothing writes it. `SkuWrite` does not take `currency`, so the only value that column can hold is the one copied from the product on insert. The product's own currency is fixed for life: `ProductWrite` requires it, and `ProductPatch` cannot change it.

The column therefore holds a second copy of a value that has no way to differ, and no constraint says the two copies must agree. A product whose SKUs are priced in another currency is unreachable through the API but representable in the database, so seed data, direct SQL, or a later write path could produce one. ADR-0027 derives `base_price` from the lowest SKU price, and that comparison means nothing unless the prices share a currency.

Two ways to close the gap:

1. Keep the column and constrain it. A composite foreign key from `sku (product_id, currency)` to `product (id, currency)` makes disagreement impossible, at the cost of a unique index on `product (id, currency)` that exists only to support the foreign key.

2. Drop the column and compose `currency` from the product on read. The read paths already require composition: `Sku.code` comes from a join to `sku_codes` (ADR-0026), and `ResolvedSku` already reads the product row for `name`, `description` and `shop_id` (ADR-0004).

## Decision

Remove `currency` from the `sku` table. Inherit`Sku.currency` and `ResolvedSku.currency` from the product row on read.

## Consequences

- The wire shape does not change. `currency` stays required on `Sku` and `ResolvedSku`, and `SkuWrite` still does not accept one.
- One currency per product is structural rather than conventional. No write path can produce currency drift between a product and its SKUs.
- ADR-0027 compares prices across a product's SKUs, and that comparison is now sound by construction.
- Composition costs nothing new, because every read that returns a SKU already holds its product.
