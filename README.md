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

The documentation in `docs/original-spec-notes.md` can be ignored. It is a historical reference to the original technical spec that contains notes out-of-scope for this build, and can be ignored during development.
