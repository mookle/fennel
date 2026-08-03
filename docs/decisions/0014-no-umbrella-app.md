# 0014 - No umbrella app

Status: Accepted

## Context

`cart` and `order` are distinct domains, both implemented using Elixir. These domains could exist as separate projects or as sub-apps under the same umbrella.

An umbrella project reduces duplication. They are particularly useful when multiple apps share config, deps, and code. `cart` and `order` share very little beyond bootstrap configuration and deployment infrastructure.

With the decision to communicate indirectly via events (ADR-0011), a "hard line" has already been drawn between the two domains.

The sub-app structure used by umbrella projects makes the domain boundary clear, but nothing other than convention and discipline prevents code from making a direct call across that boundary. 

Enforced boundaries beat convention and discipline every time.

## Decision

`cart` and `order` exist as two distinct Elixir projects, side by side in the monorepo, each with its own deps, release, Dockerfile, Helm chart, etc. 

## Consequences

- Neither project may reference the other's modules. Enforced by compiler, no discipline needed.
- Event contracts are duplicated rather than shared in a common library.
- Each service has its own build and deployment pipeline, without needing configuration to create that distinction.
- The project boundary and the deployment boundary are the same.
