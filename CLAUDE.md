# CLAUDE.md

The operating guide for any agent that works in this repository. 

## What this project is

**Fennel** is a vertical slice of a two-sided marketplace. It covers the Product, Cart and Order domains.

## Prime directives

- **The documents are the source of truth.** When your intuition and a doc disagree, the doc wins. If a doc is wrong, incomplete, or silent on something load-bearing; raise it, get a decision, and record it as an ADR before you write code.
- **Every big decision gets an ADR.** Add a new file under `docs/decisions/` with the next number, in MADR-lite form (Status, Context, Decision, Consequences) and add an index row in `docs/decisions/README.md`. Update the live documents the new decision touches.
- **An ADR is a record, not a live document.** It states one decision at one moment. When a decision changes, write a new ADR, amend the Status line of the old one, and leave its body alone. An ADR cites only records that already existed when it was written, and the Status line holds the only forward pointer. `docs/decisions/README.md` states the full rule.
- **Do not re-open a settled decision casually.** The ADRs hold the reasoning and the rejected alternatives. Read the relevant one before you propose a change.
- **`apps/product/`** is human-authored. Never create or modify files there unless explicitly asked. A deny rule in `.claude/settings.json` enforces this.

## Authoring documentation 

These rules apply to documentation, VCS-related text, error messages, release notes, and comments. It does not apply to code, identifiers, or command syntax.

### Words

- Use one name for one thing. Do not refer to the same item by two different names.
- Give each word one meaning. "fall" means to move down, not to decrease.
- No marketing adjectives: seamless, robust, powerful, cutting-edge, effortless, world-class, next-generation, revolutionary.
- Use British English spelling. "colour" not "color", "fulfilment" not "fulfillment")

### Verbs

- Active voice. "the parser reads the file", not "the file is read by the parser".
- Use a verb for an action. "analyze the log", not "perform an analysis of the log".
- No stacked auxiliaries. Not "it is important to note that this may help to improve". Write "this improves X".
- No "-ing" main verb where a simple tense works.

### Punctuation

- No em dashes.

### Structure

- One topic per paragraph, max six sentences. For steps, use a numbered vertical list, one action per item, imperative form. Put a condition before its command.
- Do not hard-wrap prose.
- Avoid run-on sentences where possible.

## Conversation

These rules apply to general chat conversation.

- Write only the requested text. No preamble, no summary, no closing remarks.
- Do not generate a recap unless explicitly asked to do so.
- Resolve questions before acting. Even when given a list of actionable items, resolve remaining questions before executing any actions.

## Naming rules

`docs/ARCHITECTURE.md` defines every domain term. Read it before you name anything. These are the distinctions to get right without a lookup:

- **A lowercase name in code format is a service, and the same word capitalised is a domain concept.** `cart` is the service, and Cart is the container it holds. `order` is the service, and Order is the unit of obligation it creates.
- **Purchase is not Order.** A Purchase is the buyer's one submission across many shops, and `cart` owns it. An Order is the per-shop unit of obligation, and `order` owns it. One Purchase becomes one Order per shop.
- **Placed is not accepted.** Placed is a buyer fact, and accepted is a seller fact. The shop is a third party that can decline, so never collapse the two.
- **OrderSku is a snapshot, not a reference.** It is immutable, and it has no status.
- **"fulfilment" is not a name in this build.** Never use it for a domain, a service or a phase. If the Order side needs a collective word, use "obligation". The word stays available in its precise sense, which is pick, pack and ship, and which this build does not model (ADR-0001).

## Load-bearing invariants

These govern the relationships between components: what each service owns, and which of them may talk to which. Every one cites the ADR that decided it, and the prime directives above say what it takes to change one.

- **Three deployables**: `product` (Go), `cart` (Elixir), `order` (Elixir). No shared database. `shop_id` and `user_id` are opaque identifiers with no backing service (ADR-0002, ADR-0001).
- **`cart` and `order` are separate Mix projects, not umbrella apps.** Neither may reference the other's modules. Each side duplicates the event envelope, and no library ever shares it (ADR-0014).
- **One database per service.** No service queries another's database (ADR-0002, ADR-0009).
- **Cart reaches Order only through RabbitMQ**, on the `purchase.submitted` event (ADR-0013).
- **Each side mints its own identifiers**: `cart` mints `purchase_id`, and `order` mints `order_id`. Neither names the other's resources. Only `purchase_id` crosses the boundary (ADR-0011).
- **`order` owns the shop split.** Cart never encodes the per-shop rule (ADR-0011).
- **Stock decrements only on `order.accepted`**, which is the moment a shop commits, not the moment a buyer submits. There is no reservation and no hold. This build accepts the oversell (ADR-0005).
- **Crystallisation**: order data is an immutable copy, never a live reference (ADR-0004).
- **Service-to-service auth** is a shared bearer token inside the cluster. There is no mTLS (ADR-0008).
- **Synchronous reads go over REST and OpenAPI**, not gRPC. **Asynchronous messages go over RabbitMQ.** Delivery is **at-least-once, with idempotent, deduped consumers** (ADR-0003, ADR-0006).

## Deployment target

Local first on **kind**, which is the default smoke-test environment. **GCP** is the cloud target, with GKE and Artifact Registry. Postgres runs in the cluster for dev, and Cloud SQL is a prod-only upgrade. Destroy the stack when it is idle (ADR-0009). **Terraform** provisions the cloud infrastructure, and **Helm** packages the services, with one chart per deployable (ADR-0010).

## Version control

- This is a **Jujutsu (`jj`) repo colocated with git**. The standard `git` commands work.
- Commit or push **only when asked**. Branch off `main` first. Do not commit directly to `main`.
- Keep authored text free of em dashes and hard wraps in a commit too, as above.

## Historical reference

`docs/original-spec-notes.md` preserves notes from the retired Coriandr spec. Never cite it, link to it, or copy from it. No live document refers to it, and the live docs stand on their own.

## Next steps

Read `./README.md` for more context and the correct reading order of the documentation.
