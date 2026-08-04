# 0016 - The build ends at `order.accepted`

Status: Accepted

## Context

Many aspects of the order domain cannot be modelled, because they depend on other domains that are out-of-scope (ADR-0001). Invoicing is one such domain. Every Order state after `accepted` rests on facts an invoicing model would own: what is owed, by whom and on what terms, what evidence marks it settled, and what follows when settlement fails. Adding a new state without the underlying model would give `order` a status that nothing can set truthfully, which is worse than not adding the state at all. **Modelling a coherent system flow is more important than modelling everything possible.**

The `accepted` state itself carries no such debt. It is a seller's decision about an Order, and `order` owns both the decision and the record of it. It is also already a published fact that other workflows can subscribe to independently (ADR-0005), so the downstream system flow can resume later without reopening the state machine.

An earlier end point is possible. The build could stop at `purchase.submitted` and leave the order domain unmodelled. Such a build performs no shop split and records no seller fact, so it demonstrates buyer intent alone. The split between buyer intent and seller obligation (ADR-0013) merits modelling, and modelling it takes both halves.

## Decision

The build ends at the `order.accepted` event.

The transition to `accepted` is automatic in this build.

## Consequences

- The Order state machine terminates at `accepted`.
- `product` decrementing stock is the only consumer of `order.accepted` in this build (ADR-0005).
- No status on an Order depends on a fact this build does not hold.
