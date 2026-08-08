# 0025 - The persistence layer maintains `updated_at`

Status: Accepted

## Context

The data held in `updated_at` is correct at the point of `INSERT` (ADR-0018). However, that correctness goes stale almost immediately as nothing enforces updates to that column when the row is written to.

A database trigger covers every write path, including bulk updates and hand-written SQL. But it is also invisible: the application layer would hold no business logic to infer the behaviour, and the table definition gives a reader no sign that the column is maintained. Further, an in-memory copy of the row would hold a stale value after every write unless the row is read back.

Application code has the opposite profile. It sits where a reader of that code can see it, and it keeps the in-memory copy of the row true. However, nothing stops a code path from omitting the write.

`updated_at` indicates a write to a row rather than a change to an entity, so the duty is not a domain concern and does not belong in domain logic. It belongs to the persistence layer that turns entities into rows.

## Decision

The persistence layer, not the domain logic, maintains `updated_at`, and no database trigger touches it.

## Consequences

- Every write path that changes a row must set `updated_at` in the same statement.
- `updated_at` is correct for any row the persistence layer has written.
- A write path that omits `updated_at` leaves the value stale, and no schema constraint catches it.
- The in-memory copy of a row is true after a write, so no write needs to read the row back to correct a timestamp.
