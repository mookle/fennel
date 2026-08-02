# order (Elixir) - Design

## Purpose

`order` owns the Order domain. It consumes `purchase.submitted`, splits it into one `Order` per shop, and runs the order state machine as far as acceptance.

`user_id` and `shop_id` are opaque identifiers that out-of-scope domains own.

## Order creation

Consuming `purchase.submitted` is the only path that creates an Order. There is no create endpoint, and nothing else writes the `orders` table.

On consume, dedupe on `purchase_id` against `processed_events`, then split the lines by `shop_id` into one Order per shop. In one transaction covering the whole message, for each shop grouping: mint an `order_id` and create the `Order`, adding its related `purchase_id`, its `OrderSku` snapshots and any addresses, then record `purchase_id` in `processed_events` once. Commit, then ack. The dedupe row commits with the orders, so a redelivery after a crash either finds the row or finds no orders, never half of each.

The rule "one Purchase becomes one Order per shop" is an Order policy and lives only here. `cart` knows nothing about it (ADR-0011).

## Identity (ADR-0011)

- **`purchase_id`**: `cart` mints it, and it arrives on the event. `order` carries it as an opaque back-reference and never mints one.
- **`order_id`**: `order` mints it when it creates the Order. It never crosses back to `cart`.

Only `purchase_id` crosses the boundary. Neither service names the other's resources.

## Model

The model keeps two entities, **item and group**, with the group (the Order) as the first-class citizen. The group is per shop, because it ties the process to one shop and one user.

- `Order` (the group): `id` (`order_id`), `purchase_id` (a back-reference), `user_id`, `shop_id`, `delivery_address` (a snapshot), `status`, `created_at`.
- `OrderSku` (the item): an immutable crystallised snapshot. **Nobody can edit it, and it has no status** (ADR-0004). Fields: `order_id`, `sku_id` (the source reference), `sku_code`, `name`, `description`, `unit_price`, `currency`, `billing_type`, `billing_period`, `quantity`, `line_shipping_cost`.
- `processed_events`: `purchase_id` (primary key), `processed_at`. The consumer's dedupe ledger, and infrastructure rather than domain: the primary key is what makes at-least-once delivery unable to act twice (ADR-0006). Each consuming service keeps its own, and `product` has an equivalent keyed on `order_id`.

## Order state machine

Implement the machine as an explicit transition module. Validate every transition.

**In this build:** `placed` to `accepted`, with `rejected` and `cancelled` as terminal branches from `placed`.

| From | Event | To | Notes |
|---|---|---|---|
| (none) | order creation | `placed` | one order per shop group. One transaction creates the order and its SKUs |
| `placed` | accept | `accepted` | the shop commits to fulfil the order. Emits `order.accepted` |
| `placed` | reject | `rejected` | the shop refuses the order. Terminal |
| `placed` | cancel | `cancelled` | the buyer cancels before acceptance. Terminal |

**Placed is a buyer fact. Accepted is a seller fact** (ADR-0005). Placement means the buyer submitted. Acceptance means the shop committed, and in a marketplace the shop is a third party that can decline. The two states stay distinct, which lets the stock decrement mean "on commitment" and not "on submission".

## Events

**Consumed:** `purchase.submitted`, from `cart`. The consumer is idempotent and dedupes on `purchase_id`.

**Emitted:** `order.accepted`, one message per accepted order, consumed by `product` for the stock decrement (ADR-0005). Delivery is at-least-once, and `product` dedupes on `order_id`.

`placed` is a state, not an event. Nothing consumes one, so `order` emits none (ADR-0005).

## Persistence

`order` owns its own Postgres database (ADR-0009). No service queries another's database (ADR-0002).

## Out of scope

Dispatch and delivery tracking, returns and exchanges, order merging, and stock reservation.
