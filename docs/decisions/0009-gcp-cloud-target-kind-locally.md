# 0009 - Cloud platform: GCP is the cloud target, and kind runs locally

Status: Accepted

## Context

The build needs a Kubernetes platform for the three deployables, both in the cloud and on a developer machine.

This is a solo-developer build, so cost and operational overhead dominate the choice. Amazon EKS charges US$0.10 per hour (about £54 to £58 per month) for the Kubernetes control plane, and there is no free tier for the cluster management fee. Worker nodes, storage and networking cost extra.

Google Kubernetes Engine charges the same US$0.10 per hour cluster management fee, but it provides a monthly US$74.40 credit per billing account. That credit covers one zonal Standard cluster or one Autopilot cluster. A single development cluster therefore incurs no management fee, although compute, storage and networking are still billed. Autopilot also removes node management, because it charges for requested pod resources rather than for user-managed node pools.

Day-to-day smoke testing does not need a cloud cluster at all. A local Kubernetes distribution such as kind runs the same workloads and avoids cloud costs during routine development.

Postgres and RabbitMQ have to run somewhere. A managed database and a managed broker both cost money and both take setup time that this build does not need to spend.

## Decision

GCP is the cloud target, with GKE for the cluster and Artifact Registry for the images. Local development runs a kind cluster, which is the default smoke-test environment. Postgres and RabbitMQ run inside the cluster in both places, and each service owns its own database.

## Consequences

- Cluster management costs are effectively zero for a single development cluster. Compute, storage and networking remain chargeable, so destroy the stack when it is idle.
- Postgres in the cluster means no managed backups and no HA in dev.
- Postgres would move outside the cluster in production, as a managed service.
- Deferred decision: should RabbitMQ also move outside the cluster in production?
- A developer needs no cloud account to run the stack.
