# 0008 - Service-to-service auth through a shared bearer token

Status: Accepted

## Context

The services talk inside the cluster. We need authentication between them, and we must not over-invest for a demo.

## Decision

Put a **shared internal bearer token** on the REST calls, and deliver the token through a Kubernetes secret. Add a network policy inside the cluster. This build uses **no mTLS**.

## Consequences

- The token is simple to configure, and a secret update rotates it.
- The token is weaker than mTLS. There is no per-service identity, and no mutual auth at the transport level. That is acceptable inside a trusted cluster for a demo. mTLS is a later hardening step.
