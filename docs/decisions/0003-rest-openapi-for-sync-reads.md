# 0003 - REST and OpenAPI for synchronous reads

Status: Accepted

## Context

This system has three synchronous cross-service read operations: catalogue query, SKU lookup, and shipping quote. They need a transport and a contract. The candidates are REST with OpenAPI, and gRPC with protobuf.

gRPC earns its cost where a caller needs typed streaming, a compact binary frame, or a generated client across many languages. This build needs none of those. It has three low-volume read paths, one producer and one consumer, and the payloads are small.

REST is first-class in both Go and Elixir, and it needs little tooling. The Elixir gRPC stack is thinner than its REST stack, so gRPC would load the extra work onto the consumer side, which is the side with the weaker support. gRPC therefore costs more than REST here and returns nothing this build can spend.

## Decision

Use **REST over HTTP and JSON for transport, with OpenAPI 3.1 as the service contract**. The OpenAPI document in `contracts/` is the source of truth. Both services generate code from it or contract-test against it.

## Consequences

- A document that is human-readable that also doubles as the contract.
- There is no typed streaming. A later path that needs it can add gRPC for that path alone, without a change to the model.
- JSON carries no decimal type, so every value that needs exact precision needs an encoding convention of its own.
