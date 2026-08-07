# 0022 - Checkout deletes the cart on submission

Status: Accepted

## Context

A Cart can exist in three states:

- `submitted`. Concrete. Duplicates a fact already immutably recorded by Purchase.
- `abandoned`. Derived. Holds no meaning in this build (ADR-0021).
- `open`. Concrete. A residual state that only exists to indicate a cart is neither `submitted` nor `abandoned`.

Nothing reads a cart after submission and no other service knows Carts exist. The buyer-facing view reads the Purchase (ADR-0011), and neither the Purchase nor the `purchase.submitted` event carries a `cart_id`.

With `submitted` gone, the column only holds one concrete value, `open`, and a one-value column carries no additional information beyond the row's existence.

## Decision

- Checkout deletes the cart row in the same transaction that inserts the Purchase.
- Cart has no status column; existence is the status.
- Cart snapshots flush on idle timeout and graceful drain only (amends ADR-0012).

## Consequences

- A live process is an open cart. A row with no live process is open until it passes the threshold (ADR-0021), and abandoned after.
- Only one source of truth persists for a buyer's intent: Purchase (ADR-0013).
- A retry will find no cart after the transaction commits, so it cannot mint a duplicate Purchase.
- A rehydration miss will create a fresh cart without needing to apply any business logic; the existence of a `submitted` status would mean every read path would need to apply a status filter.
- Abandoned rows will accumulate until an out-of-band process removes them.
