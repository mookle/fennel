# cart

## Purpose

`cart` owns the Cart domain: cart management, the checkout procedure, and the durable `Purchase` that checkout submits. It is the buyer's side of the system.

`cart` owns a buyer's **intent**. It never creates an order, never mints an `order_id`, and never touches money (ADR-0013).

`user_id` and `shop_id` are opaque identifiers that out-of-scope domains own.

## Shape

`cart` is a standalone Elixir project and deployable (ADR-0014). It shares no code and no database with `order`, and only `purchase.submitted` over RabbitMQ crosses between them. This project defines its own event envelope and payload struct, and `contracts/events.md` is the shared truth.

## Domain model

A cart organises items (potentially from multiple shops), the payment method and the delivery address. The prices it shows are advisory, because `cart` fetches them live from `product`. A price becomes a fact only when the Purchase crystallises it at submission, so a cart holds nothing worth a price lock.

- `Cart`: `id`, `user_id`, `delivery_address` (embedded), `payment_method_ref` (an opaque token), `status`, `created_at`, `updated_at`. A cart is `open`, `submitted` or `abandoned`.
- `CartItem`: `sku_id`, `shop_id`, `quantity`. These are references only. `cart` fetches the descriptive and price data from `product` for display.
- `Purchase`: the durable record of what the buyer submitted. `cart` creates it at the gate, and it is immutable after that. Fields: `id` (`purchase_id`), `user_id`, `delivery_address` (a snapshot), `lines` (the submitted snapshot: `sku_id`, `shop_id`, `sku_code`, `name`, `description`, `unit_price`, `currency`, `quantity`), `shipping` (per line and total), `submitted_at`.

The model puts `payment_method_ref` on the cart, because selection is intent. Nothing consumes it while payment is out of scope (ADR-0001).

## Memory-first lifecycle (ADR-0012)

- One process holds each active cart (`Registry` plus `DynamicSupervisor`). The state lives in the process.
- Snapshots flush to Postgres on **checkout**, on **idle timeout** (about 15 minutes) and on **graceful drain**. The drain traps SIGTERM, so a Kubernetes deploy does not lose carts.
- If a process misses on access, it rehydrates from the last snapshot. A miss with no snapshot mints a fresh cart.
- A crash between flushes loses the recent edits. This build accepts that.
- The cart table in the database is a snapshot store, not a source of truth. Nothing downstream may depend on how fresh it is.

### Purchase is the buyer-facing anchor

The "my order" read view reads the `Purchase` in this service's own database. That is deliberate. The buyer's view never reaches into the database of `order`, so the storage isolation rule holds without a cross-service read path (ADR-0002).

Purchase is also where intent becomes a record. Unlike the cart that produced it, a Purchase is durable and immutable. It is the only durable thing this service owns that outlives a session.

## Checkout procedure

Checkout is a stateless procedure, not a store. It is the move from intent to obligation, and it **ends at submission**. It does not split the cart into per-shop orders, and it mints no `order_id`. It commits one Purchase and announces it.

1. Validate the cart. Call `POST /v1/skus:batchGet` on `product` to refresh the descriptive and price data, and to confirm that each SKU still exists.
2. Check the stock before submission. Confirm that `available_quantity >= quantity` for every line. This is a synchronous read with no hold. See the ADR-0005 scaling note on oversell.
3. Get a shipping quote. Call `POST /v1/shipping/quote`, and group the lines by shop for the call. The returned costs are opaque (ADR-0007), and checkout records them per line. This grouping is for the quote only, not for order creation.
4. Mint a `purchase_id`. In one transaction, persist the **Purchase** with the crystallised submitted lines, the address and the shipping costs.
5. Publish one **`purchase.submitted`** event. One message carries the whole Purchase, and each line carries its `shop_id`. Gate the success response on the broker publisher confirm.
6. Stop the cart process.

The buyer gets "purchase submitted" and the `purchase_id`. The per-shop orders are created asynchronously.

Submission is producer-agnostic by design. A future bespoke or custom-order front end can publish `purchase.submitted` without a cart (ADR-0013).

## Product API dependency

`cart` is the only synchronous consumer of `product`. It uses the catalogue query, the SKU lookup and batch resolution, and the shipping quote. `contracts/product.openapi.yaml` holds the contract.

The test doubles for these calls come **from the examples in the contract**, not from hand-written payloads (ADR-0015). There is a `Product.Client` behaviour with Mox for the logic tests, plus a thin `Req.Test` layer that proves the real client parses real payloads.

Checkout depends on `product` being available. This build accepts that.

## Events

**Emitted:** `purchase.submitted`, one message per Purchase, consumed by `order`. See `contracts/events.md`.

**Consumed:** none. `cart` learns nothing about the orders its Purchase became. That asymmetry is intentional (ADR-0011). A real UI needs a read path or a push to show the order status.

## Persistence

`cart` owns its own Postgres database (ADR-0009). It holds the cart rows and the `purchases` table. No service queries another's database (ADR-0002).

## Out of scope

Payment method capture beyond an opaque reference, vouchers, saved carts across devices, stock reservation or holds, price locks, and cart merging. User and Shop are opaque identifiers.
