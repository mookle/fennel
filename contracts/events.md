# Async event contracts

RabbitMQ carries every event that crosses a boundary in this build (ADR-0006).

| Event | Producer | Consumer | Boundary |
|---|---|---|---|
| `order.accepted` | `order` | `product` | commitment to stock (ADR-0005) |

`placed` is an order state, not an event. Nothing consumes one, so `order` emits none.

## Delivery semantics

- **At-least-once.** A producer must not lose an event on crash. Use publisher confirms and durable queues (RabbitMQ quorum queues).
- **Idempotent consumers.** `product` dedupes `order.accepted` on `order_id`. It persists the processed identifiers and ignores the repeats. The stock decrement must be safe to receive twice.
- **Ordering** is not required. Each event is self-contained.

## Shared envelope

Every message shares this envelope. The `type` field selects the payload schema.

```json
{
  "id": "evt_01HZB2",
  "type": "order.accepted",
  "occurred_at": "2026-07-22T10:20:30Z",
  "version": 1,
  "data": { }
}
```

## Payload schemas

### `order.accepted`

The commitment fact: a shop has agreed to fulfil an order. `order` emits **one message per accepted Order**, and `product` consumes it to decrement stock (ADR-0005).

Placement is a buyer fact and carries no commitment. Acceptance is a seller fact and does carry one. Stock moves here, not at placement (ADR-0005).

- **Routing key:** `order.accepted`
- **Producer:** `order`, on the transition from `placed` to `accepted`
- **Consumer:** `product` (stock decrement)

#### `data` schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["order_id", "shop_id", "accepted_at", "lines"],
  "properties": {
    "order_id":    { "type": "string", "description": "minted by order. The dedupe key for the consumer" },
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
