# Fennel

Fennel is a vertical slice of a two-sided marketplace, inspired by a "redesign" technical spec for Coriandr, a marketplace for handmade goods I built and ran many years ago.

## Goals

This build has two goals: to improve my understanding of Go, and to experiment with agentic AI in a language I already know well - Elixir. To ensure I get the most out of the learning experience, the use of AI will be minimised where Go is being written.

## What is being built

Three domains, one deployable each:

- `product` is the catalogue (Go).
- `cart` is the memory-first cart and checkout (Elixir).
- `order` is per-shop order creation (Elixir).

The pipeline starts at adding a product to the cart and ends at order acceptance. Invoicing, payment, and fulfilment are out-of-scope.

## Start here (suggested reading order)

1. **`CLAUDE.md`**. The conventions, and the load-bearing invariants a build must not violate. Read this before you write anything.
2. **`docs/ARCHITECTURE.md`**. The canonical design: domains, service boundaries, domain language, the diagram, deployment, and data conventions.
3. **`docs/decisions/README.md`**. The index of ADRs that record every meaningful decision. Read the ones relevant to the task at hand.
4. **`docs/services/product.md`**, **`docs/services/cart.md`** and **`docs/services/order.md`**. The per-service design docs.
5. **`contracts/`**. The wire contracts the services build against: `product.openapi.yaml` for synchronous REST, and `events.md` for the asynchronous events with their JSON Schemas.
6. **`roadmap/`**. The breakdown of work remaining: `README.md` is the priority-ordered index of tasks and objectives, linking out to documents in `tasks/*` where more detail is needed.

The documentation in `docs/original-spec-notes.md` can be ignored. It is a historical reference to the original technical spec that contains notes out-of-scope for this build, and can be ignored during development.
