# Async event contracts

RabbitMQ carries every event that crosses a boundary in this build (ADR-0006).

| Event | Producer | Consumer | Boundary |
|---|---|---|---|
| `purchase.submitted` | `cart` | `order` | intent to obligation (ADR-0013) |
| `order.accepted` | `order` | `product` | commitment to stock (ADR-0005) |

`placed` is an order state, not an event. Nothing consumes one, so `order` emits none. A fuller build emits several order lifecycle events, each with more than one consumer, for example comms, analytics and seller dashboards.

## Delivery semantics

- **At-least-once.** A producer must not lose an event on crash. Use publisher confirms and durable queues (RabbitMQ quorum queues). The checkout procedure gates its success response on the confirm for the `purchase.submitted` publish.
- **Idempotent consumers.** `order` dedupes `purchase.submitted` on `purchase_id`. `product` dedupes `order.accepted` on `order_id`. Both persist the processed identifiers and ignore the repeats. The shop split and the stock decrement must both be safe to receive twice.
- **Ordering** is not required. Each event is self-contained. `order.accepted` for an order can only follow the `purchase.submitted` that created that order.

## Shared envelope

Every message shares this envelope. The `type` field selects the payload schema.

Each service defines the envelope struct on its own, and no library shares it (ADR-0014). Each side owns its own view of the wire format. This document is the shared truth.

Every amount below is a decimal string at scale 4, never a JSON number, and it always sits beside an ISO-4217 `currency` (ADR-0017). A producer pads to exactly four decimal places.

```json
{
  "id": "evt_01HZB2",
  "type": "purchase.submitted",
  "occurred_at": "2026-07-22T10:20:30Z",
  "version": 1,
  "data": { }
}
```

## Payload schemas

### `purchase.submitted`

The boundary artefact between intent (Purchase) and obligation (Order). See ADR-0013 and ADR-0011. The checkout procedure in `cart` emits **one message per Purchase**, which is the buyer's single submission across one or more shops. The message carries the customer-facing **`purchase_id`** that `cart` mints.

This is a "fat" event. It carries everything `order` needs to create the orders without a question to anyone, including the crystallised line snapshots (ADR-0004), each tagged with its `shop_id`. `order` fans the Purchase out into one Order per shop and mints each `order_id` itself. The event never carries an `order_id` (ADR-0011).

Submission is producer-agnostic. A future flow, for example a custom build order, may publish the event without a cart.

- **Routing key:** `purchase.submitted`
- **Producer:** `cart`, checkout procedure (step 5)
- **Consumer:** `order` (shop split, order creation)

The model puts the payment method selection on the cart, but the event does **not** carry it. Nothing consumes it while payment is out of scope (ADR-0001). `payment_method_ref` is where it reattaches.

#### `data` schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["purchase_id", "user_id", "delivery_address", "shipping", "lines"],
  "properties": {
    "purchase_id":        { "type": "string", "description": "minted by cart. The customer-facing reference, and the dedupe key for the consumer" },
    "user_id":            { "type": "string" },
    "delivery_address": {
      "type": "object",
      "required": ["line1", "city", "country_id"],
      "properties": {
        "name":       { "type": "string" },
        "line1":      { "type": "string" },
        "line2":      { "type": "string" },
        "city":       { "type": "string" },
        "postcode":   { "type": "string" },
        "country_id": { "type": "string" },
        "region_id":  { "type": "string" }
      }
    },
    "shipping": {
      "type": "object",
      "required": ["currency", "total", "lines"],
      "properties": {
        "currency": { "type": "string", "pattern": "^[A-Z]{3}$" },
        "total":    { "type": "string", "description": "decimal string, scale 4" },
        "lines": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["sku_id", "cost"],
            "properties": {
              "sku_id": { "type": "string" },
              "cost":   { "type": "string" }
            }
          }
        }
      }
    },
    "lines": {
      "type": "array",
      "minItems": 1,
      "description": "all lines across every shop. order groups them by shop_id",
      "items": {
        "type": "object",
        "required": ["sku_id", "shop_id", "sku_code", "name", "unit_price", "currency", "quantity"],
        "properties": {
          "sku_id":         { "type": "string" },
          "shop_id":        { "type": "string", "description": "the shop split key. order creates one Order per distinct value" },
          "sku_code":       { "type": "string" },
          "name":           { "type": "string" },
          "description":    { "type": "string" },
          "unit_price":     { "type": "string", "description": "decimal string, scale 4" },
          "currency":       { "type": "string", "pattern": "^[A-Z]{3}$" },
          "quantity":       { "type": "integer", "minimum": 1 }
        }
      }
    }
  }
}
```

#### Consumer behaviour (`order`)

1. Look up `purchase_id` in `processed_events`. If it is present, ack and stop.
2. Group `lines` by `shop_id`. In one transaction, for each group: mint an `order_id`, then create the `Order` with status `placed`, its `purchase_id`, its `OrderSku` snapshots and its address. Record `purchase_id` in `processed_events` once for the whole message.
3. Ack, then run the acceptance policy for each new order (ADR-0016).

### `order.accepted`

The commitment fact: a shop has agreed to fulfil an order. `order` emits **one message per accepted Order**, and `product` consumes it to decrement stock (ADR-0005).

Placement is a buyer fact and carries no commitment. Acceptance is a seller fact and does carry one. Stock moves here, not at placement (ADR-0005). In this build the acceptance policy accepts automatically, so the transition follows creation at once. The event name still states the rule that holds when acceptance becomes a real decision.

Acceptance is also the seam where the deferred onward process reattaches. Invoicing, payment and dispatch all hang off it (ADR-0016).

- **Routing key:** `order.accepted`
- **Producer:** `order`, on the transition from `placed` to `accepted`
- **Consumer:** `product` (stock decrement)

#### `data` schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["order_id", "purchase_id", "shop_id", "accepted_at", "lines"],
  "properties": {
    "order_id":    { "type": "string", "description": "minted by order. The dedupe key for the consumer" },
    "purchase_id": { "type": "string", "description": "a back-reference to the Purchase in cart. Opaque to product" },
    "shop_id":     { "type": "string" },
    "accepted_at": { "type": "string", "format": "date-time" },
    "lines": {
      "type": "array",
      "minItems": 1,
      "description": "one entry per OrderSku. product decrements available_quantity by quantity for each sku_id",
      "items": {
        "type": "object",
        "required": ["sku_id", "quantity"],
        "properties": {
          "sku_id":   { "type": "string" },
          "quantity": { "type": "integer", "minimum": 1 }
        }
      }
    }
  }
}
```

The payload carries only what the consumer needs. `product` decrements stock. It has no use for the prices, the names or the address. It either owns those already, or it has no business to see them.

#### Consumer behaviour (`product`)

1. Look up `order_id` in `processed_events`. If it is present, ack and stop.
2. In one transaction, decrement `sku_stock.available_quantity` by `quantity` for each line, and record `order_id` in `processed_events`.
3. Ack.

The decrement is unconditional. There is no reservation and no hold, so stock can go negative when two checkouts compete for the last unit. This build accepts the oversell (ADR-0005).
