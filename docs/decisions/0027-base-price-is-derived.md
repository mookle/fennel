# 0027 - `base_price` is derived from SKU prices

Status: Accepted

## Context

Each product has a `base_price`, intended to represent a notional "baseline" value that a variant would modify. An SKU defines the price of a given variant independently rather than modifying a baseline value.

The catalogue query returns a page of products without SKU data, because fetching that data creates an N+1 query.

Rather than represent a pricing baseline, the `base_price` property can be used as a denormalised field, but that requires a definition stating how `base_price` relates to the variant prices beneath it. Three possibilities for how `base_price` should behave:

1. A floor, below which no SKU may be priced. This would need validation on a write to both Product and Sku: a cheaper SKU would fail validation, as would raising the `base_price` above an existing SKU. No transaction spans the two endpoints, so a seller who wants a variant below the current floor must first lower the `base_price` in a separate call. Also, `base_price` as a floor guarantees that nothing is cheaper than the advertised price, not that a variant is available at that price - it would be perfectly valid to have a product with a `base_price` of £24 whose cheapest variant costs £30.

2. An independent value, authored by the seller and unrelated to the SKUs. Permissive, and as such provides no guarantees and presents little value to the consumer.

3. A value that tracks the cheapest SKU. This would require an additional write to product whenever an SKU was updated, but provides much cheaper reads on the hot path by removing the need to join against SKU data. It would also provide reliable value to consumers as the catalogue query would always return an accurate "from £xx" value.

Worth noting that regardless of option, `base_price` drift would never reach a charge. Cart reads its price from `ResolvedSku` (ADR-0004), which carries the SKU's own price, so a wrong `base_price` misleads a listing and does nothing else.

## Decision

`base_price` is the lowest `price` among the product's SKUs. The persistence layer keeps `base_price` in sync with every SKU insert, update and delete.

## Consequences

- A product with no SKUs has no `base_price`. This is a valid state - `incomplete`.
- The API presents `base_price` as read-only - `ProductWrite` and `ProductPatch` schemas drop the field.
- The catalogue query reads no SKU rows.
- "From £24" is true by construction, because an SKU exists that sells at £24.
- Adding a cheaper SKU relists the product at the lower price.
- Deleting the cheapest SKU raises the listed price.
- No write fails for a pricing reason.
- Every SKU write updates the product row, which moves the product's `updated_at` (ADR-0018).
- Comparing SKU prices is only valid while they share a currency. Sku's currency is a denormalised copy of the product's, with no writer and no constraint. As such, multi-currency-per-product is unreachable but not structurally impossible.
