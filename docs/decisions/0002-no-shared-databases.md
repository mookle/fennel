# 0002 - No service shares a database

Status: Accepted

## Context

Each domain has its own service, each service has its own deployable artifact. Should this separation extend to the persistence layer?

Both `cart` and `order` require product information, but that should not imply shared storage. The architecture must instead define how information crosses service boundaries.

Sharing a database weakens service boundaries. Schema separation relies on convention rather than enforcement; nothing structural prevents one service from reading or writing another service's tables.

Separate databases isolate credentials and privileges. Because a service can only access its own data store, the blast radius of both defects and compromised credentials is meaningfully reduced.

Each database must be provisioned, migrated, backed up and monitored independently, increasing operational overheads.

## Decision

Each service owns its own PostgreSQL instance. No service may access another service's database.

## Consequences

- Each service independently deploys, evolves, and scales its schema.
- No service can couple itself to another service's persistence model.
- Cross-service consistency is achieved through snapshotting and idempotent event processing rather than distributed transactions.
- Every database requires its own provisioning, migration, backup and operational lifecycle.
