# 0007 - The product domain computes shipping, and orders treat it as opaque

Status: Accepted

## Context

The shipping rules live with the product. Each rule covers one product, one country or region, a first-unit cost and an additional-unit cost. The cart needs the shipping costs at checkout, but it must not own the calculation.

## Decision

`product` exposes `POST /v1/shipping/quote`. `cart` sends the destination and the lines, and treats the returned costs as opaque values. These quoted values are carried unchanged into the order. The rule logic stays entirely in the product domain.

## Consequences

- Neither `cart` nor `order` carries a shipping rule. A change to the rate logic does not touch them.
- The quote endpoint creates a stable boundary around shipping. Although shipping currently lives within `product`, that boundary makes it straightforward to extract into its own domain and service later.
