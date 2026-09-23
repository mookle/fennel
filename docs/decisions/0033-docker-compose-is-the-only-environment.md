# 0033 - Docker Compose is the only environment

Status: Accepted

## Context

ADR-0009 decides upon Kubernetes as the platform and GCP as the "production" environment. ADR-0010 follows with the appropriate IaC tooling. ADR-0008 outlines a service-to-service authentication approach that is coupled with Kubernetes.

None of these decisions are objectively wrong, but they draw focus away from the stated aims of this build - to learn Go and experiment with agentic AI in Elixir.

`ARCHITECTURE.md` outlines four key aspects of this build - three deployables, no shared database, RabbitMQ between the Elixir services, and REST from `cart` to `product` (ADR-0001, ADR-0002, ADR-0006, ADR-0003). Docker is a near-universal tool at this point and widely understood. Compose can provide all four aspects.

ADR-0008 paired a shared token with a network policy. The policy limited who could call whom, and the token only proved that the caller belonged to the stack. Compose has no equivalent of a network policy, just network separation. Separate networks can limit which services reach each other, but not in one direction or per port, and they don't identify the caller.

One token per service allows the receiver to restrict access and each request to identify its source. It moves the rule on who may call whom from the network into the application.

## Decision

Docker Compose runs the stack, and is the only environment this build targets. There is no cloud target, no Kubernetes, no Terraform and no Helm. This supersedes ADR-0009 and ADR-0010.

The internal bearer token is no longer shared amongst services, but unique to each. Each service reads its token from an environment variable and is free to manage token acceptance as needed. This amends ADR-0008.

## Consequences

- The stack runs with no cloud account, no cluster and no registry. A developer needs Docker and nothing else.
- Nothing in the repository describes a deployment that does not exist. `infra/` never appears, and the live documents stop naming tools this build does not use.
- A cloud target is a fresh decision if anyone ever wants one.
- A production environment would necessitate a more secure, easily isolated service authentication mechanism, such as short-lived credentials or mTLS.
- Each receiver must maintain a list of the callers it accepts. A new caller means a config change on every existing service it calls.
