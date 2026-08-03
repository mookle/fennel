# 0012 - Cart is memory first

Status: Accepted

## Context

A cart is a mutable draft with no invariants worth protection. Lines appear, vanish and change quantity. The prices it shows are advisory, because `cart` fetches them live from `product`. Nothing in a cart is contractual. To lose one is annoying, not damaging.

Hydrating a cart from memory prevents a database round-trip.

Elixir and OTP make a process-per-cart model natural.

## Decision

Carts live in memory, one process per active cart.

## Consequences

- Cart state is periodically snapshotted to the database. Snapshots occur on checkout, idle timeout (15 minutes), and graceful drain.
- Graceful shutdown must flush active carts before the process exits.
- If a process misses on access, the cart will be rehydrated from the last snapshot.
- Cart mutation avoids database writes on the hot path and aligns naturally with OTP's process model.
- A crash between flushes loses the recent cart edits.
- The cart database table is a snapshot store, not a source of truth. Nothing downstream may depend on how fresh it is.
- Active carts require process discovery when running on multiple nodes.
