# 0023 - Sellers define SKU codes

Status: Accepted

## Context

The previous SKU format was auto-generated, consisting of a prefix generated from the product name, plus a per-shop counter, and a concatenation of each applicable option code. For example, a product called "T-Shirt" in size large, colour black could have the SKU "TS0LBLK".

A seller has no way to implement a pre-existing scheme, and many sellers will have one, likely in use via shipping labels, inventory tracking software, and product listings on other channels. An inflexible SKU scheme forces sellers to maintain a map between systems, which devalues them; an SKU's value is in its stability in the external world, across business operations.

Keeping generation as a fallback was considered. It buys zero-input SKU creation, but it comes with a high cost: a supplied-or-generated marker on every code-bearing row, regeneration rules for pre-live edits, a per-shop counter, a deterministic concatenation order requirement that would necessitate a `position` on every attribute, and a collision case where two options generate the same code. A client can suggest a code without the server carrying any of those costs.

A custom SKU code is not a pure function of the various code combinations, so code uniqueness alone cannot enforce one SKU per combination. That invariant is about identity, not naming, and needs its own constraint.

## Decision

- The seller supplies the SKU code on write
- The server never generates an SKU code.
- An SKU code must be unique per shop.
- While a product has never been live, the seller can edit an SKU code.

## Consequences

- Code collisions are resolved at SKU creation time.
- Sellers keep their external schemes, and a shared prefix across a product family costs nothing, because the server never parses a code's shape.
- Once a product has gone live, its SKUs cannot be edited; they must be deleted and rebuilt instead. This holds true regardless of the product's current state.
- One SKU per distinct option combination, per product. Code uniqueness cannot enforce this, so the SKU carries a separate constraint.
- SKU uniqueness is enforced with a conflict response on write. Endpoints must respond with a 409 on collision.
- Seed data and demos must state their codes explicitly, and a UI can suggest one client-side.
- An SKU code is a business code the seller owns, not an identifier. Nothing in the build can key on it as such.
