# 0029 - The product's currency stands in for the shop's

Status: Accepted

## Context

A seller has an operational location and as such, fixed currencies for pricing and settlement (the two are often aligned). Typically, marketplaces hold currency facts at the shop, store or account level, with products and SKUs inheriting their owner's pricing currency.

Fennel has no Shop entity, so there is nowhere to declare a shop's currency. ADR-0028 removed the copy from the `sku` table and left the product as the sole holder, but it did not say why the product holds one at all. The reason is that a product is the highest entity `product` owns that belongs to exactly one shop.

This approach leaves a gap at the catalogue level (meaning any operation across a shop's products). No write path enforces a "one shop, one currency" rule, so there is nothing to stop a shop listing a GBP product and a EUR product. That said, nothing in this build currently reads across that gap: `order` snapshots the currency onto each `OrderSku` and performs no arithmetic across lines (ADR-0004), `cart` performs no arithmetic across shops, and no document defines a per-shop total. ADR-0017 leaves multi-currency orders undefined, and this gap is the reason.

Taking a step back, there are three ways to record the shop's currency:

1. A Shop service. A fourth deployable and a fourth database to hold one column, and a re-opening of the scope ADR-0001 set.
2. A `shop_currency` table in `product`'s own database, keyed on the opaque `shop_id`. Nothing creates a shop, so the first `ProductWrite` for a `shop_id` would have to insert the row, and the shop's currency becomes an by-product of which product the seller listed first.
3. Leave the product as the holder, state that it stands in for the shop, and name the point at which the shop-level fact earns its own record.

Options 1 and 2 both levy a cost with little benefit. The existence of a currency fact at the shop level is somewhat performative, as nothing aggregates prices across products (as previously noted). Put simply - an invariant earns its enforcement when something depends on it, and nothing does yet.

## Decision

`Product.currency` is the deliberate stand-in for the shop's currency. It is fixed at creation, `ProductPatch` cannot change it, and a different currency is a different product.

One shop, one currency is a convention this build does not enforce.

The shop-level fact is deferred until something consumes it. The trigger is the first per-shop monetary aggregate: a per-shop Order total, or `cart` validating a Purchase per shop. At that point a shop-level record declares the currency, and the product inherits it on read and drops the column.

## Consequences

- A shop can hold products in different currencies.
- Per-shop aggregates stay undefined. The consumer that needs one is the consumer that pays for the shop-level record.
- A product holds a currency before it holds a price, because `base_price` is absent until an SKU exists (ADR-0027). The amount and the currency are therefore separate values on the product.
- An SKU's price copies the product's currency on read (ADR-0028).
