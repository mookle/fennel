# 0004 - Crystallise product data into an immutable OrderSku

Status: Accepted

## Context

An order shows the product and the price as they were at the time of purchase. This data must stand as a fact even after the seller edits or deletes the product.

## Decision
 
At checkout, an order creates its own immutable copy of the product data it requires. Orders reference this snapshot rather than the live product record.

## Consequences

- Checkout creates an immutable OrderSku snapshot containing the product information required to fulfil and display the order. Orders reference this snapshot rather than the live product record.
- Product edits and deletions never affect historical orders.
- The order service does not need to subscribe to product-change events.
- Some product data is intentionally duplicated in the order store.
