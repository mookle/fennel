# 0021 - Cart abandonment is derived, not stored

Status: Accepted

## Context

The Cart has an `abandoned` state, but no document defines the transition to it.

A cart process is too short-lived to set the state, because an idle timeout stops the process 15 minutes after the last touch (ADR-0012).

An out-of-band process could flip the status of old Cart rows, but a sweeper job would risk race conditions with live processes.

Nothing in this build consumes the abandonment fact: no event fires on it, no service reads it, and analytics are out of scope for this build (ADR-0001).

## Decision

- `abandoned` is a derived state, not a concrete one.
- A cart is `abandoned` when its stored status is `open` and its `updated_at` is older than the abandonment threshold.
- The abandonment threshold is 48 hours and configurable.

## Consequences

- The status column stores only `open` or `submitted`. No row ever holds `abandoned`.
- A write to an abandoned cart revives it with no explicit transition, because the write resets the age.
- There is no revival rule for rehydration to enforce.
- Nothing can query abandoned carts cheaply.
- No event fires when a cart crosses the abandonment threshold.
- A cart's `updated_at` records the flush, not the buyer's last action (ADR-0018). A flush follows the last touch by at most the idle timeout, about 15 minutes (ADR-0012), so a derived abandonment can lag the real one by that much. Against a 48-hour threshold the lag does not matter.

