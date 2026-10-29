# Architecture Decision Records

A repository of decisions whose outcomes are foundational and/or will likely have meaningful second-order effects elsewhere in the system. Not every choice and implementation detail should be recorded here - the more local, and less coupled a decision is, the less likely it is to need an ADR.

## Workflow

Add a new file under `docs/decisions/` in the format `XXXX-title-slug.md`, where `XXXX` is the next free number, zero-padded. Use a lightweight MADR format: Status, Context, Decision, Consequences. Add an index row in `docs/decisions/README.md`. Update any live documents the new decision touches. 

## Authoring rules

- **The substance of an ADR is immutable.** If a decision changes, write a new ADR. Update the status of the old ADR to refer to the record that amended or superseded it.
- **Refer to another ADR using `ADR-XXXX`.** Never a title or filename.
- **An ADR records one decision at one moment.** Beyond its status line, an ADR should contain no forward pointers. A cross-reference points backwards, to a record that already existed.
- **Never restate another ADR.** Two copies of the same rule will drift apart. Use a reference instead.
- **Only refer to another ADR if the decision at hand rests on it.** An ADR should build a case for a specific decision, not cross-reference all connected records. Cite the record that owns the fact and move on.
- **A superseded ADR stays in the tree.** There is value in recording rejected alternatives.
- **Keep the index in sync.** Update the index whenever an ADR is added, a status changes, or when a filename or title has been updated.
- **Do not re-open an accepted decision casually.** The ADRs hold the reasoning and the rejected alternatives. Read the relevant one before you propose a change.

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](0001-partial-scope-three-domains.md) | Partial scope: three domains across three deployables | Accepted |
| [0002](0002-no-shared-databases.md) | No service shares a database | Accepted |
| [0003](0003-rest-openapi-for-sync-reads.md) | REST and OpenAPI for synchronous reads | Accepted |
| [0004](0004-order-sku-crystallisation.md) | Crystallise product data into an immutable OrderSku | Accepted |
| [0005](0005-stock-decrement-via-event.md) | Stock decrement via event | Accepted |
| [0006](0006-event-broker-rabbitmq.md) | Event broker: RabbitMQ | Accepted |
| [0007](0007-shipping-in-product-domain.md) | The product domain computes shipping, and orders treat it as opaque | Accepted |
| [0008](0008-service-auth-bearer-token.md) | Service-to-service auth through a shared bearer token | Accepted (amended by ADR-0033) |
| [0009](0009-gcp-cloud-target-kind-locally.md) | Cloud platform: GCP is the cloud target, and kind runs locally | Superseded by ADR-0033 |
| [0010](0010-terraform-provisions-helm-packages.md) | Infrastructure as code: Terraform provisions, and Helm packages | Superseded by ADR-0033 |
| [0011](0011-one-cart-many-orders.md) | One cart can create many orders | Accepted |
| [0012](0012-cart-memory-first.md) | Cart is memory first | Accepted (amended by ADR-0022) |
| [0013](0013-buyer-intent-seller-obligation.md) | Buyer intent ends at purchase; seller obligation begins at order | Accepted |
| [0014](0014-no-umbrella-app.md) | No umbrella app | Accepted |
| [0015](0015-contract-derived-test-doubles.md) | Contract-derived test doubles for the product API | Accepted |
| [0016](0016-build-ends-at-order-accepted.md) | The build ends at `order.accepted` | Accepted |
| [0017](0017-money-is-fixed-scale-decimal.md) | Money is NUMERIC(15,4), and a fixed-scale decimal string on the wire | Accepted |
| [0018](0018-updated-at-not-null-equals-created-at.md) | `updated_at` is NOT NULL, and equals `created_at` at INSERT | Accepted |
| [0019](0019-status-history-when-transitions-are-facts.md) | A status lives in a history table when its transitions are facts | Accepted |
| [0020](0020-billing-types-removed.md) | Billing types removed: every purchase is immediate | Accepted |
| [0021](0021-cart-abandonment-is-derived.md) | Cart abandonment is derived, not stored | Superseded by ADR-0022 |
| [0022](0022-cart-deleted-on-submission.md) | Checkout deletes the cart on submission | Accepted |
| [0023](0023-sellers-define-sku-codes.md) | Sellers define SKU codes | Accepted |
| [0024](0024-entity-ids-uuidv7.md) | Entity ids are UUIDv7, minted by the owning service | Accepted |
| [0025](0025-persistence-layer-maintains-updated-at.md) | The persistence layer maintains `updated_at` | Accepted |
| [0026](0026-sku-code-is-an-allocation.md) | A SKU code is an allocation, not a column | Accepted |
| [0027](0027-base-price-is-derived.md) | `base_price` is derived from SKU prices | Accepted |
| [0028](0028-currency-lives-on-the-product.md) | Currency lives on the product, not the SKU | Accepted (amended by ADR-0032) |
| [0029](0029-product-currency-stands-in-for-shop.md) | The product's currency stands in for the shop's | Accepted |
| [0030](0030-currency-must-hold-iso-4217-code.md) | Currency must hold an ISO 4217 currency code | Accepted |
| [0031](0031-absent-amount-is-a-separate-type.md) | An absent amount is a separate type | Accepted |
| [0032](0032-amount-and-currency-are-one-value.md) | An amount and its currency are one value | Accepted |
| [0033](0033-docker-compose-is-the-only-environment.md) | Docker Compose is the only environment | Accepted |
