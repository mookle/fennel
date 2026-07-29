# 0001 - Partial scope: three domains across three deployables

Status: Accepted

## Context

The original platform (re)design covered nine modules. The stated goals of learning Go and experimenting with agentic AI don't require a full build, just a vertical slice to best exercise/demonstrate those two aspects. The agentic AI experiment should be written in a known language, so that the AI's output can readily be verified. Right now, that means Elixir.

## Decision

The journey from cart to order covers standard REST API fetching and state management, both of which map well to the strengths of Go and Elixir respectively. Focus on the order journey, building three domains, mapped to **three deployables**:

- **Product**: catalogue lookup and query. Products, SKUs including code generation, attributes and options, labels, and shipping cost calculation. Deployable: **product** (Go).
- **Cart**: cart management. The memory-first cart, line items, address and payment selection, and the checkout procedure that creates a `Purchase`. Deployable: **cart** (Elixir).
- **Order**: order creation. Consume `purchase.submitted`, create an Order, and run the Order state machine. Deployable: **order** (Elixir).

## Consequences

- The surface area is small and clear, and it is fast to stand up.
- The platform is not expected to run end to end.
- All other domains (e.g. user, shop, image, invoicing, payment) are out of scope. Where a flow needs an enitity from one of these other domains, it will use an opaque identifier. 
- "fulfilment" is not a name in this build. In its precise sense it means pick, pack and ship, which this build does not model. In its loose sense it names a phase, and a phase groups by when work happens rather than by what a service owns, which makes it a poor boundary and an invitation to orchestration.
