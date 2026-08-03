# cart

## Purpose

`cart` owns the Cart domain: cart management and the checkout procedure. It is the buyer's side of the system.

`user_id` and `shop_id` are opaque identifiers that out-of-scope domains own.

## Domain model

A cart organises items (potentially from multiple shops), the payment method and the delivery address. The prices it shows are advisory, because `cart` fetches them live from `product`. A price becomes a fact only when the Purchase crystallises it at submission, so a cart holds nothing worth a price lock.

- `Cart`: `id`, `user_id`, `delivery_address` (embedded), `payment_method_ref` (an opaque token), `status`, `created_at`, `updated_at`. A cart is `open`, `submitted` or `abandoned`.
- `CartItem`: `sku_id`, `shop_id`, `quantity`. These are references only. `cart` fetches the descriptive and price data from `product` for display.
- `Purchase`: the durable record of what the buyer submitted. `cart` creates it at the gate, and it is immutable after that. Fields: `id` (`purchase_id`), `user_id`, `delivery_address` (a snapshot), `lines` (the submitted snapshot: `sku_id`, `shop_id`, `sku_code`, `name`, `description`, `unit_price`, `currency`, `billing_type`, `billing_period`, `quantity`), `shipping` (per line and total), `submitted_at`.

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

## Events

**Emitted:** `purchase.submitted`, one message per Purchase, consumed by `order`. See `contracts/events.md`.

**Consumed:** none. `cart` learns nothing about the orders its Purchase became. That asymmetry is intentional (ADR-0011). A real UI needs a read path or a push to show the order status.

## Persistence

`cart` owns its own Postgres database (ADR-0009). It holds the cart rows and the `purchases` table. No service queries another's database (ADR-0002).

## Out of scope

Payment method capture beyond an opaque reference, vouchers, saved carts across devices, stock reservation or holds, price locks, and cart merging. User and Shop are opaque identifiers.
