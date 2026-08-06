# 0020 - Billing types removed: every purchase is immediate

Status: Accepted

## Context

`billing_type` and `billing_period` sat on Product, travelled on ResolvedSku, crossed the boundary between `cart` and `order` on `purchase.submitted` and came to rest on OrderSku. Together, the two columns provided a lookup key for the differing payment flows a flexible marketplace must permit.

The build ends at `order.accepted` (ADR-0016), the canonical event for seller commitment that any number of workflows can subscribe to (ADR-0005). No invoicing or payment domain exists in this build (ADR-0001), so nothing reads either `billing_type` or `billing_period`.

## Decision

Remove `billing_type` and `billing_period` from the build.

## Consequences

- Every purchase is considered to be immediate.
- Unneeded data is not modelled or transmitted.
- Reintroducing the fields will require a migration on Product, ResolvedSku, and OrderSku; an event schema change that will touch both `cart` and `order`; and an update to the OpenAPI product schema that will impact both `product` and `cart`.
