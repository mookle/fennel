# order (Elixir) - Design

## Purpose

`order` owns the Order domain. It creates Orders and runs the order state machine.

`user_id` and `shop_id` are opaque identifiers that out-of-scope domains own.

## Model

The model keeps two entities, **item and group**, with the group (the Order) as the first-class citizen. The group is per shop, because it ties the process to one shop and one user.

- `Order` (the group): `id` (`order_id`), `user_id`, `shop_id`, `delivery_address` (a snapshot), `status`, `created_at`.
- `OrderSku` (the item): `order_id`, `sku_id`, `sku_code`, `name`, `description`, `unit_price`, `currency`, `quantity`, `line_shipping_cost`.

## Out of scope

Dispatch and delivery tracking, returns and exchanges, order merging, and stock reservation.
