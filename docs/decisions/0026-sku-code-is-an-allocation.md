# 0026 - A SKU code is an allocation, not a column

Status: Accepted

## Context

An SKU code's value is in its stability in the external world: shipping labels, inventory software and listings on other channels all hold it. A code that has reached that world must never be reused to point at different goods.

SKU deletion has three paths: a delete endpoint, the removal of its related attribute option, and the removal of its parent product. Furthermore, once a product has been live its SKUs freeze, which turns an SKU edit into a delete -> (re)create operation (ADR-0023), so deletion will likely be far from a rare event.

One option is to maintain a registry of dead codes, a table written to on deletion and consulted alongside the live SKU rows by the per-shop uniqueness check. That creates two sources of truth for the question "is this code taken?", and it holds only if every future delete path remembers to write to the registry. Forgetting once reassigns a dead code silently, which is exactly the failure the registry exists to prevent.

A dead code registry would also require `Sku` to own a copy of `shop_id` in order to utilise a plain unique index on `sku`; Postgres cannot span the join to the `product` table to get that data.

A code is more than a business fact about an SKU: it is an allocation with lifecycle rules beyond the SKU itself.

## Decision

- SKU codes are allocations, stored in their own table (`sku_codes`), with a unique index on `(shop_id, code)`.
- An allocation is claimed by its insertion into `sku_codes`.
- The code string lives only on the allocation.
- While the product has never been live, an allocation can change or be released: a pre-live edit updates the row in place, and deleting a never-live SKU deletes its allocation with it. From first go-live the allocation is permanent, and deleting an SKU, directly or by cascade, leaves it standing.
- A retired code is an allocation with no live SKU. The unique index on `sku_codes` keeps it from reassignment.
- An allocation is an internal construct, not an entity. It has an integer id.

## Consequences

- An insertion conflict is the per-shop uniqueness failure. 
- No two SKUs can ever share one allocation.
- There is one source of truth for the question "is this code taken?", for live and dead codes alike.
- After go-live, reuse prevention is the default rather than a step: deleting an SKU preserves its code by doing nothing, so no future delete endpoint can forget it. Releasing a pre-live allocation is the one deletion that takes a second explicit write, and a code the outside world has never seen costs nothing if that write goes missing.
- `Sku` drops `code` and references its allocation uniquely.
- The per-shop constraint lives on the table that carries `shop_id`, and `sku` never gains the column.
- SKU creation gains one insert and one foreign key.
- The allocation table grows forever, because nothing deletes a row after go-live.
- `product` must compose `Sku.code` and `ResolvedSku.sku_code` using a join when sending data on the wire.
