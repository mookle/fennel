# Architecture Decision Records

The big decisions for Fennel. The format is lightweight MADR: Status, Context, Decision, Consequences.

A superseded ADR stays in the tree. It records the reasoning and the rejected alternatives, and both stay useful when someone picks up the deferred work. The number is the stable reference. A cross-reference uses `ADR-NNNN`, never a filename. A slug tracks the current title, so a retitled ADR gets a new filename and a new link in this table.

**An ADR records one decision at one moment, and later work never edits it.** A cross-reference points backwards, to a record that already existed. The one forward pointer is the Status line, which names whatever superseded or amended this decision, and this table repeats it. Nothing else in the body points forward.

**A fact belongs in an ADR only if the decision rests on it.** Cite the record that owns the fact and move on. Never restate another ADR's rule, because two copies of one rule drift apart. Never report what later happened to this decision, because that is the job of the record that changed it.

When a decision changes, write a new ADR. Amend the Status line of the old one, and leave its body alone.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-partial-scope-three-domains.md) | Partial scope: three domains across three deployables | Accepted |
| [0002](0002-no-shared-databases.md) | No service shares a database | Accepted |
| [0003](0003-rest-openapi-for-sync-reads.md) | REST and OpenAPI for synchronous reads | Accepted |
| [0004](0004-order-sku-crystallisation.md) | Crystallise product data into an immutable OrderSku | Accepted |
| [0005](0005-stock-decrement-via-event.md) | Stock decrement via event | Accepted |
| [0006](0006-event-broker-rabbitmq.md) | Event broker: RabbitMQ | Accepted |
| [0007](0007-shipping-in-product-domain.md) | The product domain computes shipping, and orders treat it as opaque | Accepted |
| [0008](0008-service-auth-bearer-token.md) | Service-to-service auth through a shared bearer token | Accepted |
| [0009](0009-gcp-cloud-target-kind-locally.md) | Cloud platform: GCP is the cloud target, and kind runs locally | Accepted |
| [0010](0010-terraform-provisions-helm-packages.md) | Infrastructure as code: Terraform provisions, and Helm packages | Accepted |
| [0011](0011-one-cart-many-orders.md) | One cart can create many orders | Accepted |
| [0012](0012-cart-memory-first.md) | Cart is memory first | Accepted |
| [0013](0013-buyer-intent-seller-obligation.md) | Buyer intent ends at purchase; seller obligation begins at order | Accepted |
| [0014](0014-no-umbrella-app.md) | No umbrella app | Accepted |
| [0015](0015-contract-derived-test-doubles.md) | Contract-derived test doubles for the product API | Accepted |
| [0016](0016-build-ends-at-order-accepted.md) | The build ends at `order.accepted` | Accepted |
| [0017](0017-money-is-fixed-scale-decimal.md) | Money is NUMERIC(15,4), and a fixed-scale decimal string on the wire | Accepted |
| [0018](0018-updated-at-not-null-equals-created-at.md) | `updated_at` is NOT NULL, and equals `created_at` at INSERT | Accepted |
