# CLAUDE.md

The operating guide for any agent that works in this repository. 

## What this project is

**Fennel** is a vertical slice of a two-sided marketplace. It covers the Product, Cart and Order domains.

## Prime directives

- **Tests > code > documentation.** The efficacy of a design decision cannot be evaluated until it has been practised in code, and the code needs tests to be proved behaviourally correct. Building documentation ahead of time risks baking in assumptions, and wasting effort better spent exercising the real system. When working, provable code exists, then record the decisions behind it.
- **Code is the source of truth.** If code and documentation disagree, the code is the representation that matters. Note that "truth" and "correct" are not the same thing - confirm the implementation is sound before taking action to close the gap. 
- **Avoid restating documentation.** There are times where it makes sense to summarise content held elsewhere, but a single source of truth prevents drift and the resulting confusion. Use a reference where possible.
- **Dead documentation is harmful.** It creates confusion and erodes the reader's trust. Documentation that frequently comes adrift from the implementation may be misplaced or poorly structured. Raise the issue for discussion should you encounter any such prose.
- **Every big decision gets an ADR.** Read `docs/decisions/README.md` for the ADR authoring rules, important steps to take before proposing a new decision, and a definition of what constitutes "big".

## Naming rules

`docs/ARCHITECTURE.md` defines every domain term. Read it before you name anything. These are the distinctions to get right without a lookup:

- **A lowercase name in code format is a service, and the same word capitalised is a domain concept.** `cart` is the service, and Cart is the container it holds. `order` is the service, and Order is the unit of obligation it creates.
- **Purchase is not Order.** A Purchase is the buyer's one submission across many shops, and `cart` owns it. An Order is the per-shop unit of obligation, and `order` owns it. One Purchase becomes one Order per shop.
- **Placed is not accepted.** Placed is a buyer fact, and accepted is a seller fact. The shop is a third party that can decline, so never collapse the two.
- **OrderSku is a snapshot, not a reference.** It is immutable, and it has no status.
- **An entity's own field is unqualified, and a field holding another entity's value carries that entity's name.** `Sku.code` is bare, and `ResolvedSku.sku_code`, `OrderSku.sku_code` and the `purchase.submitted` line's `sku_code` are qualified because they copy it. The two spellings are not a drift to tidy up.
- **"fulfilment" is not a name in this build.** Never use it for a domain, a service or a phase. If the Order side needs a collective word, use "obligation". The word stays available in its precise sense, which is pick, pack and ship, and which this build does not model (ADR-0001).

## Load-bearing invariants

These govern the relationships between components: what each service owns, and which of them may talk to which.

- **Three deployables**: `product` (Go), `cart` (Elixir), `order` (Elixir). No shared database. `shop_id` and `user_id` are opaque identifiers with no backing service (ADR-0002, ADR-0001).
- **`cart` and `order` are separate Mix projects, not umbrella apps.** Neither may reference the other's modules. Each side duplicates the event envelope, and no library ever shares it (ADR-0014).
- **One database per service.** No service queries another's database (ADR-0002).
- **Cart reaches Order only through RabbitMQ**, on the `purchase.submitted` event (ADR-0013).
- **Each side mints its own identifiers**: `cart` mints `purchase_id`, and `order` mints `order_id`. Neither names the other's resources. Only `purchase_id` crosses the boundary (ADR-0011).
- **`order` owns the shop split.** Cart never encodes the per-shop rule (ADR-0011).
- **`order` never reads `product`.** Everything it needs arrives crystallised on the event. `cart` is the only synchronous consumer of `product` (ADR-0004, ADR-0002).
- **Crystallisation**: order data is an immutable copy, never a live reference (ADR-0004).
- **The build ends at `accepted`.** There is no invoicing, no payment and no settlement. All three reattach at `order.accepted` (ADR-0016).
- **Stock decrements only on `order.accepted`**, which is the moment a shop commits, not the moment a buyer submits. There is no reservation and no hold. This build accepts the oversell (ADR-0005).
- **The test doubles for the product API come from `contracts/product.openapi.yaml`**, never from a hand-written payload (ADR-0015).
- **Synchronous reads go over REST and OpenAPI**, not gRPC. **Asynchronous messages go over RabbitMQ.** Delivery is **at-least-once, with idempotent, deduped consumers** (ADR-0003, ADR-0006).
- **Service-to-service auth.** Each service identifies itself via a bearer token (ADR-0008, ADR-0033).

## Data conventions

These govern the representation of values: what a column's type is, and what a field looks like on the wire. They bind exactly as hard as the invariants above, and they change the same way. `docs/ARCHITECTURE.md` holds the full set and the reasons. These are the ones where the obvious default is the wrong one:

- **Money** is `NUMERIC(15,4)`, and never a float. JSON carries the amount as a **string**, never as a number, padded to exactly four decimal places on output. Always pair it with an ISO-4217 `currency`, and never do arithmetic across two currencies (ADR-0017).
- **`updated_at` is `NOT NULL` and means the last write.** It equals `created_at` on an untouched row, and it is never null on the wire. Never treat it as a flag for whether the row changed, and never use `updated_at != created_at` as that flag. `created_at` marks the insertion. Use `updated_at` only where a row change carries no historical meaning (ADR-0018).
- **A status lives in a history table when its transitions are facts**, meaning a transition carries a reason, an actor, a time the row's own timestamps do not hold, or a value that can recur. Otherwise the status is a column on the entity row. The shape of the state graph is not the test (ADR-0019).
- **Every entity id is a UUIDv7**, and the owning service mints it in application code. The column keeps a `DEFAULT uuidv7()` for seed data and direct SQL, and application code never relies on it. JSON carries it as a plain lowercase UUID string. Join and bookkeeping tables keep their natural keys, and `shop_id` and `user_id` stay opaque strings with no constrained shape (ADR-0024). An SKU code is a business code, not an identifier (ADR-0023).

## Deployment target

Docker Compose only (ADR-0033).

## Version control

- This is a Jujutsu (`jj`) repo colocated with git. The standard `git` commands work.
- Use worktrees. Do not work in the main tree unless explicitly requested to do so.
- Use a feature branch workflow. Branch off `main`, commit changes, and raise a PR when the feature is complete. Do not commit directly to `main`.
- Read the style guidance in `docs/AUTHORING.md` before writing a commit message or PR description.

## Historical reference

`docs/original-spec-notes.md` preserves notes from the retired Coriandr spec. Never cite it, link to it, or copy from it. No live document refers to it, and the live docs stand on their own.

## Next steps

Read `docs/AUTHORING.md` to understand the conventions you must follow when writing prose. Then read the main project `./README.md` for more context and detail.
