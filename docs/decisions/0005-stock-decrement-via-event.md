# 0005 - Stock decrement via event

Status: Accepted

## Context

Stock lives in `product`. `cart` is concerned with building buyer intent; placing an order is a **buyer** fact. In a marketplace, the seller is a third party who can decline an order (stock sold offline, unacceptable shipping destination, holiday, etc); commitment to fulfil an order is a **seller** fact.

Until a seller has accepted an order, stock should not be decremented.

Seller acceptance is a significant business event. Significant business events should be published, particularly when they mark a boundary between domains.

The order service could notify `product` synchronously via REST or asynchronously by publishing an event. A synchronous call would couple seller acceptance to the availability of product, requiring both services to be healthy before the workflow can complete. The stock update is not part of the seller's transaction and can safely occur shortly afterwards. An event therefore reduces temporal coupling while preserving the business sequence.

## Decision

At the point a seller agrees to fulfil an order, the order service emits an `order.accepted` event. The product service decrements stock in response.

## Consequences

- `order` never decrements stock directly. 
- `order` and `product` remain loosely coupled. A short outage of `product` does not block seller acceptance because the event remains in the broker until it is processed.
- Consumers require a mechanism to ensure idempotency (e.g. a `processed_events` table).
- Two concurrent checkouts for the last unit can both pass the pre-payment check and oversell.  A hold or reservation mechanism is future work.
- `order.accepted` becomes the canonical business event marking the transition from buyer intent to seller commitment. Other workflows (payment capture, notifications, fulfilment, analytics) can subscribe to it independently.
