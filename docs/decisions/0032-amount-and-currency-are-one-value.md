# 0032 - An amount and its currency are one value

Status: Accepted

## Context

Every amount pairs with an ISO-4217 currency, and arithmetic between two currencies has no defined result (ADR-0017). The contract carries the two as sibling fields: `price` beside `currency` on `Sku` and `ResolvedSku`, `base_price` beside `currency` on `Product`, and one `currency` over the `lines` and `total` of a shipping quote. The Go type in `product` follows the contract and holds the amount alone, with the currency a separate value on the entity.

Four reasons held the amount and the currency apart, and none of them survives inspection:

1. A product with no SKUs has no `base_price` (ADR-0027), so a product holds a currency with no amount beside it. That is absence of the whole value, and a separate type already holds it (ADR-0031). It says nothing about whether a present amount can lack a currency.
2. A product holds its currency from creation because it stands in for the shop's (ADR-0029). That is a fact about the product, and it is the reason the product carries a currency at all. It is not a reason a price carries none.
3. `SkuWrite` takes no currency, and the `sku` row has no currency column (ADR-0028). But the service reads the product on both paths anyway: the write reads it to check the SKU's options against the product's attributes, and the read composes `currency` from it. The currency is at hand wherever an amount is.
4. The contract's flat shape is the contract, and `cart` consumes it. The contract is this build's to change, and `cart`'s doubles derive from it rather than transcribe it (ADR-0015).

What the split cost is that the domain type was not the domain concept. A price is an amount in a currency, and a type that holds the amount alone can neither refuse arithmetic across two currencies nor tell a reader which currency it is in. Every caller carried the check the type could not, and ADR-0017's rule that arithmetic between two currencies has no result was a caveat in a comment rather than an error at the call.

Two ways to hold the value:

1. The amount alone, with the currency on the entity beside it. The type matches the column and the flat wire shape, and each caller pairs the two.
2. One value with both halves, composed at each boundary from what that boundary holds. The type matches the concept, the wire shape carries the pair as an object, and the column keeps the amount alone.

## Decision

An amount and its currency form one value. Absence is absence of the whole value (ADR-0031), and a value with one half is not admitted.

On the wire the value is an object that carries both halves, and a response carries both or neither. A request that supplies an amount with no currency carries the bare amount, and the service composes the value from the product's currency (ADR-0028). `SkuWrite.price` is the one such field.

A read composes the value from the amount column and the product's currency. The columns stay as they are, and the product keeps its own currency as the shop's stand-in (ADR-0029).

Arithmetic and comparison across two currencies fail at the call.

## Consequences

- The wire shape changes. `Product.base_price`, `Sku.price`, `ResolvedSku.price` and the shipping quote's `cost` and `total` become objects, and `currency` stops being a sibling field on `Sku`, `ResolvedSku` and the shipping quote. This amends ADR-0028, whose consequence that the wire shape does not change no longer holds. `Product.currency` stays.
- `cart`'s doubles pick the change up from the contract (ADR-0015). The `purchase.submitted` event is `cart`'s and does not change (ADR-0014).
- Storage does not change. One composite type exists for reads, so a query can hand the pair to the application as one column.
- The wire, the read and memory carry one shape, the pair. The column alone holds the scalar.
- A product with a `base_price` carries its currency twice on the wire: once as the shop's stand-in, and once inside the value.
- Nothing in `product` can produce two currencies at one call, because every amount it handles comes from one product. The mismatch error guards a case the build cannot reach, and it is there so that the first consumer that aggregates across products meets an error rather than a wrong sum.
