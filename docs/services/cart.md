# cart

## Purpose

`cart` owns the Cart domain: cart management and the checkout procedure. It is the buyer's side of the system.

`user_id` and `shop_id` are opaque identifiers that out-of-scope domains own.

## Domain model

A cart organises items (potentially from multiple shops), the payment method and the delivery address. The prices it shows are advisory, because `cart` fetches them live from `product`.

- `Cart`: `id`, `user_id`, `delivery_address` (embedded), `payment_method_ref` (an opaque token), `status`, `created_at`, `updated_at`. A cart is `open`, `submitted` or `abandoned`.
- `CartItem`: `sku_id`, `shop_id`, `quantity`. These are references only. `cart` fetches the descriptive and price data from `product` for display.

The model puts `payment_method_ref` on the cart, because selection is intent. Nothing consumes it while payment is out of scope (ADR-0001).

## Persistence

`cart` owns its own Postgres database (ADR-0009). No service queries another's database (ADR-0002).

## Out of scope

Payment method capture beyond an opaque reference, vouchers, saved carts across devices, stock reservation or holds, price locks, and cart merging. User and Shop are opaque identifiers.
