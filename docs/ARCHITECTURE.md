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
| **Order** | Order creation and acceptance: consume `purchase.submitted`, split by shop, create Orders, run the order state machine, emit `order.accepted`. | `order` | Elixir |


**The build ends at order acceptance** (ADR-0016). Invoicing, payment and settlement are out of scope. A more complete build reattaches them at the `order.accepted` event.

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


**The build ends at order acceptance** (ADR-0016). Invoicing, payment and settlement are out of scope. A more complete build reattaches them at the `order.accepted` event.

### Domain language, continued

**The concepts.**

- **Cart**: the container for an intended order. It owns line items, addresses and payment method. It can hold line items from more than one shop.
- **Checkout**: the procedure inside `cart` that adds an address and a payment method to a Cart, then submits it. It is a UX process, not a domain. It ends when it mints a `purchase_id` and publishes `purchase.submitted`. It never creates orders.
- **Purchase**: the buyer's single submitted act, across many shops. `cart` creates it at submission and owns it. It is durable, immutable, and the customer-facing reference that `purchase_id` keys. This is where intent becomes a record.
- **Order**: the per-shop unit of obligation. `order` creates one Order per shop from a Purchase, mints each `order_id`, and carries `purchase_id` as a back-reference. It then runs the order state machine.
- **OrderSku**: the immutable crystallised snapshot of a SKU at the moment an order is created. Nobody can edit it, and it has no status (ADR-0004). `docs/services/order.md` lists its fields.

**The events.**

- **`purchase.submitted`**: the boundary event. One message per Purchase, from `cart` to `order`, over RabbitMQ (ADR-0013).
- **`order.accepted`**: the write-back event. `order` emits it when a shop commits, and `product` consumes it to decrement stock. It is also the seam where deferred invoicing and payment reattach (ADR-0005, ADR-0016).

**The facts.**

- **Placed**: a buyer fact, the record that the buyer submitted. `order` sets it on each Order it creates (ADR-0005).
- **Accepted**: a seller fact. It occurs when a shop commits to fulfil an Order. In a marketplace the shop is a third party that can decline, so the two moments are distinct (ADR-0005).
- **Rejected**: a seller fact. It occurs when a shop does not commit to an Order. The reasons vary, for example repeated declined payment attempts, or an unrealistic custom order. In practice this state is rare. Most shops accept incoming orders automatically.

**Order writes to Product.** The relationship is asymmetric, and the two Elixir services hold different halves of it (ADR-0014, ADR-0002). `order` writes back to `product` asynchronously and once. On acceptance it emits `order.accepted`. `product` consumes that event and decrements stock (ADR-0005). This is the only write into the product domain, and it is never a direct call. `order` never reads `product`. Everything it needs arrives crystallised on the `purchase.submitted` event.

```
   ┌──────────┐  purchase.submitted   ┌──────────┐   order.accepted   ┌───────────┐
   │   cart   │ ────── (broker) ────▶ │  order   │ ─── (broker) ────▶ │  product  │
   │ (Elixir) │                       │ (Elixir) │                    │   (Go)    │
   └──────────┘                       └──────────┘                    └───────────┘
        │                                                                   ▲
        └──────── sync REST: catalogue, sku lookup, shipping quote ─────────┘

   cart:    memory-first cart, checkout,   order:   shop split, orders,   product: products, skus,
            durable Purchase                        state machine to               attributes, labels,
                                                    accepted                       shipping, stock
```

See `contracts/product.openapi.yaml` for the REST contract and `contracts/events.md` for the asynchronous contracts. The ADRs indexed at `docs/decisions/README.md` record the reason for each choice and the alternatives rejected. The inline `(ADR-NNNN)` tags in this document point to the relevant record.

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
- CI builds one container image per service. CI also runs contract tests against `contracts/` before it deploys. The tests validate the OpenAPI examples against their own schemas. They check that the handlers in `product` return schema-conforming responses. They also derive the test doubles for `cart` from those examples instead of from hand-written payloads (ADR-0015).
- Each service owns its **own Postgres database**, at version 18 or later, which the `uuidv7()` default on every id column needs (ADR-0024). The databases run in the cluster for dev, and Cloud SQL is a prod-only upgrade. No service reaches another service's database.

## Data conventions

These rules bind every schema and every payload in the build, and this section is the source of truth for them. The set predates this build, carried over from the original spec, and ADR-0017, ADR-0018, ADR-0019 and ADR-0024 have since decided the four that were load-bearing and unrecorded.

- Text is UTF-8. Prefer correct i18n and sorting over micro-optimisation.
- Money is `NUMERIC(15,4)`. JSON carries it as a **string** to keep the precision, padded to exactly four decimal places on output. Always pair it with an ISO-4217 `currency`, and never do arithmetic across two currencies. Nothing in this build rounds, because every operand is at scale 4 and every quantity is an integer. ADR-0017 holds the rule for the day that changes.
- `created_at` marks the insertion. `updated_at` is `NOT NULL` and means the last write, so it equals `created_at` on an untouched row (ADR-0018). It is never a flag for whether the row has changed, and `updated_at != created_at` is not that flag either. Use `updated_at` only where row changes carry no historical meaning. The persistence layer maintains the column, and no database trigger touches it, so every write path that changes a row sets it in the same statement (ADR-0025).
- Status history: a status lives in a history table when its transitions are facts, meaning a transition carries a reason, an actor, a time the row's own timestamps do not hold, or a value that can recur. Otherwise the status is a column on the entity row. The shape of the state graph is not the test (ADR-0019).
- Every entity id is a UUIDv7, and the owning service mints it in application code. Every id column also carries `DEFAULT uuidv7()`, which catches seed data and any other direct SQL, and which application code never leans on. JSON carries an id as a plain lowercase UUID string. The scope is entity ids only: join and bookkeeping tables keep their natural keys, and `shop_id` and `user_id` stay opaque strings with no constrained shape (ADR-0024). A SKU code is a business code, not an identifier (ADR-0023).
- An entity's own field is unqualified, and a field holding another entity's value carries that entity's name. `Order.id` and `Sku.code` are bare, while `OrderSku.order_id`, `OrderSku.sku_code` and every `ResolvedSku` field are qualified, because they reference or copy what another entity owns.
