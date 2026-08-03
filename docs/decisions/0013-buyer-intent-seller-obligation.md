# 0013 - Buyer intent ends at purchase; seller obligation begins at order

Status: Accepted

## Context

The boundary between `cart`, checkout and `order` was unclear while checkout was responsible for payment execution. Two principles resolve this:

- Buyer intent and seller obligation are different things.
- Payment is not required for an order to exist.

## Decision

A Purchase represents **buyer intent** and is owned by `cart`. An Order represents **seller obligation** and is owned by `order`.

Checkout is a stateless procedure that ends with `cart` persisting a Purchase and publishing `purchase.submitted`. `purchase.submitted` is the boundary artifact between buyer intent and seller obligation.

`order` creates Orders only by consuming `purchase.submitted`.

## Consequences

- Each durable artefact has exactly one writer.
- Checkout returns immediately with a `purchase_id`; order creation is asynchronous.
- The checkout procedure can be completed regardless of the availability of the `order` service.
- Order creation is **producer-agnostic**; any workflow may publish `purchase.submitted`.
- Payment execution can evolve independently of checkout.
