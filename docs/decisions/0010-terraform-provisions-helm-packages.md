# 0010 - Infrastructure as code: Terraform provisions, and Helm packages

Status: Accepted

## Context

ADR-0009 picks the platform and leaves the tooling open. The build still requires a repeatable, infrastructure-as-code path to that platform.

Two jobs are distinct. Provisioning creates the cloud resources: the cluster, the registry and the networking. Packaging turns each service into a deployable Kubernetes release. One tool that does both would tie the local path to the cloud path, and local development has no cloud resources to provision.

## Decision

Terraform provisions the cloud infrastructure. Helm packages the services and deploys them onto Kubernetes, with one chart per deployable. The same charts run on the local cluster and in the cloud.

## Consequences

- Each service requires its own Helm chart, and the broker needs one too.
- The charts are the same locally and in the cloud, so a chart change is exercised locally before it reaches the cloud.
- Provider-specific resources stay inside the Terraform modules, which keeps any future provider change local.
- The local path runs Helm alone, because there is nothing for Terraform to provision there.
- Two tools means two sets of state and two failure modes.
