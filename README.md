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

## Start here

1. **`CLAUDE.md`**. Designed for AI agents, but a useful summary for all. Contains conventions, load-bearing invariants a build must not violate, and guidance on how to interact with the project. Agents **must** read this first.
2. **`docs/ARCHITECTURE.md`**. The canonical design: domains, service boundaries, domain language, the diagram, deployment, and data conventions.
3. **`docs/services/product.md`**, **`docs/services/cart.md`** and **`docs/services/order.md`**. The per-service design docs.
4. **`contracts/`**. The wire contracts the services build against: `product.openapi.yaml` for synchronous REST, and `events.md` for the asynchronous events with their JSON Schemas.
5. **`docs/decisions/README.md`**. The index of the architectural decision records (ADRs) and the rules around their creation.
6. **`roadmap.md`**. A working document / scratchpad for co-ordinating ongoing work between authors.
7. **`docs/AUTHORING.md`.** The authoring style guide, relevant when writing prose.

The documentation in `docs/original-spec-notes.md` can be ignored during feature development. It is a historical reference to the original technical spec that contains notes out-of-scope for this build.
