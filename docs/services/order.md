# order (Elixir) - Design

## Purpose

`order` owns the Order domain. It consumes `purchase.submitted`, splits it into one `Order` per shop, and runs the order state machine as far as acceptance. `order` owns **obligation** (ADR-0013).

The build ends when a shop accepts an order (ADR-0016). Invoicing, payment, dispatch and completion are future work. They reattach at `order.accepted`.

`user_id` and `shop_id` are opaque identifiers that out-of-scope domains own.

## Shape

`order` is a standalone Elixir project and deployable (ADR-0014). It shares no code and no database with `cart`, and only `purchase.submitted` over RabbitMQ crosses between them. This project defines its own event envelope and payload struct, and `contracts/events.md` is the shared truth.

## Order creation

Consuming `purchase.submitted` is the only path that creates an Order. There is no create endpoint, and nothing else writes the `orders` table.

On consume, dedupe on `purchase_id` against `processed_events`, then split the lines by `shop_id` into one Order per shop. In one transaction covering the whole message, for each shop grouping: mint an `order_id` and create the `Order`, adding its related `purchase_id`, a `placed` status in `OrderStatusHistory`, its `OrderSku` snapshots and any addresses, then record `purchase_id` in `processed_events` once. Commit, then ack. The dedupe row commits with the orders, so a redelivery after a crash either finds the row or finds no orders, never half of each.

The rule "one Purchase becomes one Order per shop" is an Order policy and lives only here. `cart` knows nothing about it (ADR-0011).

## Identity (ADR-0011)

- **`purchase_id`**: `cart` mints it, and it arrives on the event. `order` carries it as an opaque back-reference and never mints one.
- **`order_id`**: `order` mints it when it creates the Order. It never crosses back to `cart`.

Only `purchase_id` crosses the boundary. Neither service names the other's resources.

## Model

The model keeps two entities, **item and group**, with the group (the Order) as the first-class citizen. The group is per shop, because it ties the process to one shop and one user.

- `Order` (the group): `id` (`order_id`), `purchase_id` (a back-reference), `user_id`, `shop_id`, `delivery_address` (a snapshot), `created_at`. It carries no status column.
- `OrderSku` (the item): an immutable crystallised snapshot. **Nobody can edit it, and it has no status** (ADR-0004). Fields: `order_id`, `sku_id` (the source reference), `sku_code`, `name`, `description`, `unit_price`, `currency`, `quantity`, `line_shipping_cost`.
- `OrderStatusHistory`: `order_id`, `status`, `reason`, `created_at`. The history goes in a separate table because these transitions are facts: acceptance and rejection are seller decisions, and each one carries a `reason` that a marketplace must be able to produce later (ADR-0019). An Order's current status is the latest row here. `order` exposes no read that filters orders by status, so caching status directly on `Order` gains little.
- `processed_events`: `purchase_id` (primary key), `processed_at`. The consumer's dedupe ledger, and infrastructure rather than domain: the primary key is what makes at-least-once delivery unable to act twice (ADR-0006). Each consuming service keeps its own, and `product` has an equivalent keyed on `order_id`.

## Order state machine

Implement the machine as an explicit transition module. Validate every transition, and record it in `OrderStatusHistory`.

**In this build:** `placed` to `accepted`, with `rejected` and `cancelled` as terminal branches from `placed`.

| From | Event | To | Notes |
|---|---|---|---|
| (none) | order creation | `placed` | one order per shop group. One transaction creates the order and its SKUs |
| `placed` | accept | `accepted` | the shop commits to fulfil the order. Emits `order.accepted` |
| `placed` | reject | `rejected` | the shop refuses the order. Terminal |
| `placed` | cancel | `cancelled` | the buyer cancels before acceptance. Terminal |

**Placed is a buyer fact. Accepted is a seller fact** (ADR-0005). Placement means the buyer submitted. Acceptance means the shop committed, and in a marketplace the shop is a third party that can decline. The two states stay distinct, which lets the stock decrement mean "on commitment" and not "on submission".

### Acceptance policy

The transition is automatic for now. It lives behind one **acceptance policy** seam, an auto-accept rule that always returns true, instead of inline code at the call site (ADR-0016). A real seller-driven or rules-driven acceptance then replaces one function.

`rejected` is implemented and reachable, even though nothing triggers it yet. A state machine with one path is a queue. The branch is what makes acceptance a decision.

### Deferred

`dispatched`, `completed`, `returned` and `merged` are out of scope for this build (ADR-0016). So are returns and exchanges, changes to order items after placement, and order merging. They extend the machine past `accepted`, and they change nothing below it.

Two more statuses are named here so a fuller build reuses the words, and both stay deferred:

- `superseded`: closes an Order that a corrected reissue replaces (an address change, a discount). It is the mechanism behind change-after-placement, and it needs a link from the closed Order to its replacement, which is the one model change on this list.
- `reversed`: the post-`accepted` counterpart of `cancelled`. It stays a separate status because cancelling before commitment undoes nothing, while reversing after it must undo the stock decrement and, in a fuller build, the payment. One `reversed` state covers withdrawal by the buyer, the seller or both: the actor belongs on the history row, so a `void` or `closed` variant would encode in the status value what an `actor` column on `OrderStatusHistory` records better (ADR-0019).

## Events

**Consumed:** `purchase.submitted`, from `cart`. The consumer is idempotent and dedupes on `purchase_id`.

**Emitted:** `order.accepted`, one message per accepted order, consumed by `product` for the stock decrement (ADR-0005). Delivery is at-least-once, and `product` dedupes on `order_id`.

`placed` is a state, not an event. Nothing consumes one, so `order` emits none (ADR-0005).

## Persistence

`order` owns its own Postgres database (ADR-0009): `orders`, `order_skus`, `order_status_history` and `processed_events`. No service queries another's database (ADR-0002).

## Out of scope

Invoicing, payment, transactions and settlement (ADR-0016). Dispatch and delivery tracking, returns and exchanges, order merging, and stock reservation. The item and group model stays general on purpose, so invoices and subscriptions can attach at `order.accepted` later.
