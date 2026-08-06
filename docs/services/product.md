# product (Go) - Design

## Purpose

`product` owns the product catalogue: products, SKUs, attributes and their options, labels, stock, and shipping cost calculation. It serves synchronous reads to `cart`. It consumes `order.accepted` events and decrements stock. This service exposes the API surface only, the seller front end is out of scope.

`shop_id` is an opaque identifier that an out-of-scope Shop domain owns.

## Domain footprint

"Product" here is one service that covers what a fuller marketplace separates into three domains:

- **Catalogue**: products, SKUs, attributes, options, labels. The bulk of the model. Read-heavy data - written by sellers, read constantly by all.
- **Inventory**: stock levels. Write-heavy data - written by an event consumer, potentially contended, and the only thing another service can mutate.
- **Pricing**: `base_price` on the product and `price` on the SKU. Both are static here, whereas a real Pricing domain grows promotions, campaigns, price history and currency conversion, none of which are in scope.

The package layout can mirror the three domains (`internal/catalogue`, `internal/inventory`, `internal/pricing`). Keep the types concrete, and declare each interface at the point of use. Ports and adapters between three packages in one binary is ceremony this build does not need.

## Domain model

### Product

The customer-facing listing. A product lists only when at least one SKU exists; short of that it stays `incomplete`. The sellable units (SKUs) come from combinations of attribute options, and a product with no attributes carries the single empty combination.

Fields: `id`, `shop_id`, `name` (a descriptor, not `title`), `description`, `currency`, `base_price`, `billing_type`, `billing_period`, `created_at`, `updated_at`.

`billing_type` and `billing_period` select the payment flow a flexible marketplace must permit: `immediate` for a one-off purchase, or `recurring` with a period for a subscription.

**Statuses**:

- `incomplete`: below the listing threshold above. Every product starts here, because a SKU attaches only after the product row exists.
- `active`: listed, searchable and purchasable.
- `hidden`: listable, but the shop has opted out, for example during a holiday.
- `archived`: delisted and hidden from the shop's normal view. Not editable. Restorable to another status.
- `reported`: flagged as inappropriate and delisted pending review. The shop can still edit it. It moves to `deleted`, or back to the status it held before the report.
- `deleted`: permanently delisted by the platform. The shop cannot reverse this.

The public catalogue query returns `active` products only.

### ProductStatusHistory

Every status transition writes one row, in the same transaction as the write it accompanies. `reported` can recur, so the history is the only place the earlier occurrences survive (ADR-0019).

Fields: `product_id`, `status`, `reason`, `created_at`.

The history is what restores a reported product. A return from `reported` needs the status the product held before it, and one column cannot hold a value that recurs.

`reason` is a string, and this build does not fix its format. A fuller build that adds report categories or moderator notes decides the shape then.

This table introduces several performance costs when querying products. Requiring a join means a query must resolve the latest history per product before it can filter, and pagination can no longer cut to a page. It also gives up a partial index on a common query, `(shop_id) WHERE status = 'active'`. One fix would be to cache `status` on the Product row, but until performance has been measured, any fix is premature optimisation.

### Attribute and AttributeOption

Attributes are optional: a product has 0-n attributes, and each attribute has 1-n options. A shop adds attributes to a product, for example Size. Each attribute has options, for example S, M and L. The shop turns the set of *applied* options into SKUs. The product records which attributes and options exist. It does not generate every combination.

- `Attribute`: `id`, `product_id`, `name`.
- `AttributeOption`: `id`, `attribute_id`, `name`.

### SKU

The sellable, orderable unit. The shop creates each SKU by hand from a chosen combination of attribute options, then sets the available quantity. A SKU carries one option for each attribute the product holds, and one SKU exists per distinct combination. On a product with no attributes the combination is empty, so the product carries at most one SKU.

Fields: `id`, `product_id`, `code`, `price`, `currency`, `created_at`, `updated_at`.

The server generates `code` from a prefix derived from the product name, a per-shop counter, and the code of each applied option. A product called "T-Shirt" in size large, colour black gets `TS0LBLK`.

The applied option combination lives in the `sku_options` join table (`sku_id`, `option_id`), one row per applied option, and no rows for an option-less SKU. The API presents it as `option_ids`. A join table rather than an id array keeps the foreign keys real, and it answers "which SKUs use this option", which is the question SKU dynamism asks (see the notes below).

Stock is **not** a column here. See SkuStock below.

Notes:

- The end user cannot edit a SKU.
- SKUs are dynamic: if the shop removes an attribute option, the SKUs that used it are deleted, not archived. No endpoint removes an option in this build, so the rule is latent, but the boundary already absorbs it. `POST /v1/skus:batchGet` returns the missing ids so `cart` can fail the affected lines, and an accepted order needs nothing from the row because `order_sku` is the crystallised copy (ADR-0004).

### SkuStock

Stock lives in a separate, narrow table keyed by `sku_id`. It is not a column on `sku`.

- `SkuStock`: `sku_id` (primary key, foreign key to `sku`), `available_quantity`, `created_at`, `updated_at`.

`created_at` marks the first stock write for the SKU, and `updated_at` the last (ADR-0018).

The reason is the write profile, not tidiness. SKU rows are cold: every catalogue request reads the code and the price, and nothing rewrites them after creation. Stock is hot: the `order.accepted` consumer writes it on every acceptance. In Postgres an update rewrites the whole row, so on a combined table every decrement leaves a dead copy of the SKU row, and unless the update stays HOT it also touches the row's indexes. `sku_stock` carries no index beyond its primary key, which keeps the decrement HOT-eligible and keeps the vacuum churn off the table the read path depends on.

A read that needs price and quantity together, such as `ResolvedSku`, joins the two tables.

### Label

Labels are the platform's categorisation tags, and they drive the site categories. A label is platform-wide and shared across shops, and a product can carry many. Each label has **aliases**, which stop fragmentation: "Jewellery" and "Jewelry" resolve to one canonical label.

- `Label`: `id`, `name` (canonical), `slug`. The tag itself, one row per category.
- `LabelAlias`: `id`, `label_id`, `alias`. An alternative spelling that resolves to its label.
- `ProductLabel`: `product_id`, `label_id`. The assignment join: one row attaches one label to one product.

### ShippingRule

`product` owns the shipping rules, and they are opaque to the callers. A shop defines which countries and regions a product ships to, and any extra cost beyond the first unit.

- `ShippingRule`: `id`, `product_id`, `country_id` (nullable), `region_id` (nullable), `first_unit_cost`, `additional_unit_cost`, `currency`.

Country and region rules are optional, which rules out a clean composite key. `id` is therefore a surrogate key. Store both country and region as positive ids.

When more than one rule matches a destination, the most specific rule wins: a region rule beats a country rule. The quote endpoint is opaque to its callers, so this resolution rule lives entirely in `product`.

## Tax

One tax type, no rates and no logic. Tax is out of scope. Do not model tax tables.

## API surface

`contracts/product.openapi.yaml` holds the full contract. There are two audiences.

**Consumed by `cart` (must stay stable):**

- `GET /v1/products`: catalogue query and search. The filters are `shop_id`, `label`, `q` and `status`. The results are paginated. The public read returns `active` products only.
- `GET /v1/products/{id}`: product detail with the attributes, the options and the SKUs.
- `GET /v1/skus/{id}` and `POST /v1/skus:batchGet`: the `ResolvedSku` payloads that cart and checkout consume. Each response is self-contained: `sku_code`, `product_id`, `shop_id`, `name`, `description`, `price`, `currency`, `available_quantity`, and the resolved options.
- `POST /v1/shipping/quote`: takes `{ destination, items[] }` and returns the per-line and total shipping cost. The caller (checkout) treats the result as opaque.

**Product-domain management (this service's own surface):**

- `POST /v1/products` and `PATCH /v1/products/{id}`: create and update a product, including a validated status transition.
- `POST /v1/products/{id}/attributes`: add an attribute with its options.
- `POST /v1/skus`: create a SKU from a combination of options, with an initial `available_quantity`.
- `GET /v1/labels` and `POST /v1/labels`: list the canonical labels, and create one with optional aliases.

The contract holds nothing else. Deletion, option removal, later stock corrections and shipping-rule management have no endpoint in this build; seed data writes the `shipping_rules` rows directly.

## Event consumption

`product` consumes `order.accepted` from the broker (see `contracts/events.md`). On receipt, decrement `sku_stock.available_quantity` for each `sku_id` in the payload.

- Delivery is **at-least-once**. The handler must be **idempotent** on `order_id`. Persist the processed order ids, and ignore the duplicates.
- The decrement is the only change another service can trigger, and it is never a direct call (ADR-0002).
- Stock moves on **acceptance**, the point where a shop commits to fulfil the order (ADR-0005). It does not move on placement. Placement is a buyer fact and carries no commitment.
- **Deferred (scaling note):** there is no reservation and no temporary hold. Two near-simultaneous checkouts for the last unit can both pass the synchronous pre-submission check and oversell. This is acceptable for the demo. A hold or reservation mechanism is future work (see ADR-0005).

## Persistence

`product` owns its own Postgres database (ADR-0009). The suggested tables mirror the model above: `products`, `product_status_history`, `attributes`, `attribute_options`, `skus`, `sku_stock`, `sku_options`, `labels`, `label_aliases`, `product_labels`, `shipping_rules`, and `processed_events` for idempotency. No service queries another's database (ADR-0002).

## Not in scope

Seller UI, images (a product image is an opaque identifier at most), reporting, tax, and reservations.
