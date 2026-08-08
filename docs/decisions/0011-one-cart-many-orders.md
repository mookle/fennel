# 0011 - One cart can create many orders

Status: Accepted

## Context

A cart can contain products from multiple shops. A buyer submits once during checkout. Something must split a purchase to produce per-shop orders. This split could be owned by either `cart` or `order` domains.

A Purchase is the buyer's view of the transaction. It captures the cart as submitted and may contain items from multiple sellers. `cart` owns Purchases, which are the terminal artifact of checkout and represent the buyer's intent.

An Order is the seller's view of the transaction. `order` owns Orders, creating one per seller so each can independently reject or accept and fulfil the sale.

Identity follows the same line. One "order" concept and one `order_id` conflates the buyer's single act with the per-shop unit, so each side needs an identifier of its own. The alternative is that `cart` pre-allocates each `order_id`, or that checkout emits one message per shop. Both make `cart` mint identity for a resource `order` owns, and both put the per-shop rule on the buyer's side of the boundary.

## Decision

`cart` emits `purchase.submitted` after creating the Purchase. `order` consumes this event and creates one Order per shop.

Each side mints its own identifier. `cart` mints `purchase_id` at submission, and `order` mints each `order_id` when it creates the Order. `order` carries `purchase_id` as an opaque back-reference. Neither service names the other's resources, and only `purchase_id` crosses the boundary.

## Consequences

- `cart` remains responsible for buyer workflow; `order` owns seller workflow from order creation onwards.
- The service that creates the resource also owns it.
- One Purchase can produce several Orders, each with its own lifecycle.
- The Purchase row exists the instant the buyer submits, so the customer-facing reference is durable regardless of `order` availability.
- The `purchase.submitted` event must carry `purchase_id` in order to uphold idempotency guarantees.
- The buyer's "my order" read view reads the Purchase in the database of `cart`, and never crosses into the database of `order` (ADR-0002).
- Changing the shop-split rule later, for example to split one shop's lines into two Orders, touches `order` alone.
