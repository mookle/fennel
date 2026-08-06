# 0019 - A status lives in a history table when its transitions are facts

Status: Accepted

## Context

Every entity that carries a status needs to decide whether to encode that status as a column on the entity row, or in a separate history table. Both options hold the current value, so the decision must rest on other domain requirements.

The shape of the state graph is not a factor; the journey from state A to state B is not interesting in and of itself. For example, a Cart moves linearly from `open` -> `submitted` or `abandoned` and an Order also moves linearly from `placed` -> `accepted` or `rejected`. There is little value in a historical state record of a Cart, but great value in the same for an Order.

The entity row approach can hold two facts: the current status value, and when the
row was last written. It holds them for the most recent write only, and it dates
the transition only when that transition was the last write.

So the key determinant is whether the domain needs more information about a transition than the row can hold.

## Decision

An entity gets a status history table when its state transitions carry facts the domain requires. Otherwise status is a column on the entity row.

A transition carries a fact when it records:

- **When** it happened, beyond the row's notion of "last written".
- **Why** it happened.
- **Who** caused it.

A status that can **recur** settles the first on its own. The row keeps the latest arrival and discards the earlier ones, so a status that can be entered, left and entered again always has a "when" the row cannot hold.

Where none of those applies, the current value and the row's own timestamps hold everything the domain needs.

## Consequences

- The rules defined by this decision must be applied to an entity whenever a new status value is added.
- A change to the shape of an entity's state graph does not require a reapplication of the rules.
