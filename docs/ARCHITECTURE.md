# Fennel - Architecture

## Context

Fennel is a vertical slice of a two-sided marketplace with a focus on **order creation**. It covers the lookup and query part of a Product domain, plus a Cart to Order pipeline that ends at order acceptance.

## Scope: three domains, three deployables

A service owns its code, its deployment, its persistence and its published contracts.

One domain per deployable, one to one (ADR-0001)

| Domain | Concern in this build | Deployable | Language |
|---|---|---|---|
| **Product** | Catalogue lookup and query: products, SKUs, attributes and options, labels, stock, shipping cost calculation. This is the read surface that `cart` uses. Seller management UIs are out of scope. | `product` | Go |
| **Cart** | Cart management: memory-first cart, line items across shops, address selection, and the checkout procedure that submits a `Purchase`. | `cart` | Elixir |
| **Order** | Order creation: consume `purchase.submitted`, create Orders, run the order state machine. | `order` | Elixir |

### Domain language

This section defines each term once, and it is the source of truth for all of them. `CLAUDE.md` carries only the naming rules that follow from these definitions. The per-service docs carry the field-level detail.

**The services.**

- **`product`**: the Go service. It owns the Product domain: products, SKUs, attributes and options, labels, stock, and shipping cost calculation. It straddles what a fuller build splits into Catalogue, Inventory and Pricing. The schema draws only the Catalogue and Inventory line (`sku_stock`). Seller management UIs are out of scope.
- **`cart`**: the Elixir service that owns intent. It holds the memory-first cart, the checkout procedure, and the durable Purchase that checkout submits. It is the only synchronous consumer of `product`.
- **`order`**: the Elixir service that owns obligation. It creates one Order per shop from a Purchase, and runs the order state machine to acceptance. It never reads `product`.

The word **fulfilment** is deliberately not a name in this build (ADR-0001). In its precise sense it means pick, pack and ship, which this build does not model. In its loose sense it names a phase, not a domain. Where the Order side needs a collective word, use **obligation** from ADR-0013.

## Service boundaries

Three services connect. The services share nothing: no common database and no identity service. `shop_id` and `user_id` are opaque identifiers.

`cart` and `order` are **separate Mix projects**, not co-located under the same umbrella app. Each side duplicates the event envelope and the payload structs instead of sharing a library. Each side owns its own view of the wire format, and `contracts/events.md` is the shared truth (ADR-0014).

Each service owns its own database, and no service reaches into another's (ADR-0002). Cross-service consistency comes from snapshotting and idempotent processing rather than from distributed transactions.

**Cart to Order (intent to obligation).** Checkout ends at submission. It mints a customer-facing `purchase_id` and publishes one `purchase.submitted` fact to the broker. `order` consumes that fact, splits it into one Order per shop, and mints each `order_id`. From that point `order` owns everything durable. Neither service names the other's resources, and only `purchase_id` crosses (ADR-0011).

**Cart reads Product.** `cart` reads from `product` synchronously over REST. `cart` is the only consumer of `product`. Checkout crystallises everything it reads into an immutable snapshot, so a later product edit never changes a historical order (ADR-0004).

### Domain language, continued

**The concepts.**

- **Cart**: the container for an intended order. It owns line items, addresses and payment method. It can hold line items from more than one shop.
- **Checkout**: the procedure inside `cart` that adds an address and a payment method to a Cart, then submits it. It is a UX process, not a domain. It ends when it mints a `purchase_id` and publishes `purchase.submitted`. It never creates orders.
- **Purchase**: the buyer's single submitted act, across many shops. `cart` creates it at submission and owns it. It is durable, immutable, and the customer-facing reference that `purchase_id` keys. This is where intent becomes a record.
- **Order**: the per-shop unit of obligation. `order` creates one Order per shop from a Purchase, mints each `order_id`, and carries `purchase_id` as a back-reference. It then runs the order state machine.
- **OrderSku**: the immutable crystallised snapshot of a SKU at the moment an order is created. Nobody can edit it, and it has no status (ADR-0004). `docs/services/order.md` lists its fields.

**The events.**

- **`purchase.submitted`**: the boundary event. One message per Purchase, from `cart` to `order`, over RabbitMQ (ADR-0013).

**The facts.**

- **Placed**: a buyer fact, the record that the buyer submitted. `order` sets it on each Order it creates (ADR-0005).
- **Accepted**: a seller fact. It occurs when a shop commits to fulfil an Order. In a marketplace the shop is a third party that can decline, so the two moments are distinct (ADR-0005).
- **Rejected**: a seller fact. It occurs when a shop does not commit to an Order. The reasons vary, for example repeated declined payment attempts, or an unrealistic custom order. In practice this state is rare. Most shops accept incoming orders automatically.

## Monorepo layout

```
.
├── apps/
│   ├── product/                # Go service (Product domain)
│   ├── cart/                   # Elixir service
│   └── order/                  # Elixir service
├── contracts/                  # source of truth for cross-service wire formats
│   ├── product.openapi.yaml
│   └── events.md               # JSON Schemas per event
├── infra/
│   ├── terraform/              # cluster and cloud resources, per env
│   │   ├── modules/
│   │   └── envs/{dev,prod}/
│   └── helm/                   # one chart per deployable
│       ├── product/
│       ├── cart/
│       ├── order/
│       └── rabbitmq/           # or a charted dependency
└── docs/
    ├── ARCHITECTURE.md         # this file
    ├── services/
    │   ├── product.md
    │   ├── cart.md
    │   └── order.md
    ├── original-spec-notes.md  # historical, ignored during development
    └── decisions/              # ADRs
```

## Deployment (ADR-0009, ADR-0010)

- **Local first**: a **kind** cluster is the default smoke-test environment. It runs Postgres and RabbitMQ in the cluster.
- **GCP** is the cloud target. **Terraform** provisions GKE (Autopilot, or zonal with spot nodes), Artifact Registry, and the networking. Destroy the stack when it is idle.
- **Helm** packages each service for the cluster. There is one chart per deployable, plus one for the broker.
- Each service owns its **own Postgres database**. The databases run in the cluster for dev, and Cloud SQL is a prod-only upgrade. No service reaches another service's database.

## Data conventions

These rules bind every schema and every payload in the build, and this section is the source of truth for them. The set predates this build, carried over from the original spec, and the load-bearing ones have no record behind them yet.

- Text is UTF-8. Prefer correct i18n and sorting over micro-optimisation.
- `created_at` marks the insertion. `updated_at` stays NULL until something writes the row, so one column doubles as the flag for whether the row has ever changed.
- An entity's own field is unqualified, and a field holding another entity's value carries that entity's name. `Order.id` and `Sku.code` are bare, while `OrderSku.order_id`, `OrderSku.sku_code` and every `ResolvedSku` field are qualified, because they reference or copy what another entity owns.
