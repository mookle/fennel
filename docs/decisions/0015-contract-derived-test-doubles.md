# 0015 - Contract-derived test doubles for the product API

Status: Accepted

## Context

`cart` reads the product API synchronously through the catalogue query, SKU lookup and shipping quote. Its tests therefore need a stand-in for a service they do not own.

JSON fixtures are simple, but they duplicate the API contract. As the contract evolves they silently drift from reality, hiding mistakes in payload shape, required fields and data representation.

A hand-written mock service avoids duplicating fixtures, but introduces another artefact that must evolve alongside the real service.

ADR-0003 makes contracts/product.openapi.yaml the source of truth for the synchronous API. Test doubles should therefore be derived from that contract rather than maintained independently.

## Decision

Derive all HTTP-level test doubles from the OpenAPI contract.

- Business logic tests use a mocked Product.Client behaviour (Mox).
- HTTP client tests replay response examples extracted directly from contracts/product.openapi.yaml.
- Local development and kind smoke tests use the real Go product service.
- CI validates the OpenAPI examples against their schemas and verifies that product returns schema-conforming responses.
- If a runnable mock service is required, generate it from the OpenAPI document rather than maintaining one by hand.

## Consequences

- The OpenAPI document becomes the single source of truth for API payloads used in tests.
- Contract drift becomes a build failure rather than a silent divergence.
- Adding or changing an endpoint requires updating its examples in the contract.
- Behavioural changes that preserve the schema (for example, rounding or business rules) are not detected. Consumer-driven contract testing remains future work.
- The Elixir test suite gains a small amount of tooling to extract examples from the OpenAPI document.
