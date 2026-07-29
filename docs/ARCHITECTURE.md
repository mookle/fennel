# Fennel - Architecture

## Context

Fennel is a vertical slice of a two-sided marketplace with a focus on **order creation**. It covers the lookup and query part of a Product domain, plus a Cart to Order pipeline that ends at order acceptance.

## Scope: three domains, three deployables

A service owns its code, its deployment, its persistence and its published contracts.

One domain per deployable, one to one (ADR-0001)

| Domain | Concern in this build | Deployable | Language |
|---|---|---|---|
| **Product** | Catalogue lookup and query: products, SKUs, attributes and options, labels, stock, shipping cost calculation. This is the read surface that `cart` uses. Seller management UIs are out of scope. | `product` | Go |
| **Cart** | Cart management: memory-first cart, line items across shops, address selection, and the checkout procedure that submits a `Purchase`. | `cart` | Elixir |
| **Order** | Order creation: consume `purchase.submitted`, create Orders, run the order state machine. | `order` | Elixir |

### Domain language

This section defines each term once, and it is the source of truth for all of them. `CLAUDE.md` carries only the naming rules that follow from these definitions. The per-service docs carry the field-level detail.

**The services.**

- **`product`**: the Go service. It owns the Product domain: products, SKUs, attributes and options, labels, stock, and shipping cost calculation. It straddles what a fuller build splits into Catalogue, Inventory and Pricing. Seller management UIs are out of scope.
- **`cart`**: the Elixir service that holds the cart and the checkout procedure that submits it.
- **`order`**: the Elixir service that creates Orders and runs the order state machine.

**The concepts.**

- **Cart**: the container for an intended order. It owns line items, addresses and payment method. It can hold line items from more than one shop.
