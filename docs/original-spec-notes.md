# Original spec notes (historical reference)

Preserved from the retired Coriandr Technical Specification PDF. **Non-normative**: where anything here conflicts with `docs/ARCHITECTURE.md`, `docs/services/`, `contracts/`, or `docs/decisions/`, those win. Content already fully transferred into the live specs (product/inventory model, cart, order flow and statuses, SKU code generation, shipping rules, data conventions) is not repeated; this file preserves the rest: platform vision, out-of-scope modules, design rationale, and the original doc's open questions.

## Platform vision and aim

- Coriandr was a two-sided marketplace; the spec was a redesign of a live system, taking ~80 tables to ~60 while increasing capability. Changes to the spec doc were expected to precede code changes.
- The codebase was to be divided into service domains enabling a family of interrelated products: Coriandr/marketplace, Ecommerce platform, Inventory management, Image management, Invoicing. Only the first two (possibly three) were considered achievable; the marketplace and ecommerce platform were the primary aims.
- Spec-level @todos never resolved: preferences, fee structures, a Coriandr abstract.

## Event layer (original design)

The platform core would use an event layer so each domain could implement contextual logic without coupling. Initial release used Symfony's in-memory event system; no messaging infrastructure was planned (the snapshot build supersedes this with RabbitMQ, ADR-0006).

Lifecycle rules: events triggered within controllers, grabbed by listeners which invoke domain logic as single calls, listeners acting as thin wrappers. **Events must not be triggered from within the domain/service layer**, to (a) keep domain logic ignorant of how it's invoked, (b) keep listeners free of logic, (c) prevent a "vertically stacked" event layer where one event triggers cascades across the system.

System events catalogue (by bundle):

- **UserBundle** — user: create, register, verify, delete, suspend, ban, login, logout; message: sent, read
- **ShopBundle** — shop: create, add_admin, remove_admin, delete; sale: start, end; holiday: start, end
- **InventoryBundle** — product: create, hide, show, relist, sale, delete, view
- **ImageBundle** — image: upload, delete, crop, tag, assign
- **FulfillmentBundle** — order: place, pay, ship, dispatch, cancel, complete
- **ForumBundle** — post: create, delete, edit; topic: create, delete, edit, read, reply; board: read
- **CartBundle** — cart: add_item, remove_item, checkout, payment

## User module (out of scope in this build)

A support module reusable by each product, not a product itself. Users, customers, and buyers are the same concept. **Users are never deleted.** Any user who has bought from a shop is considered a user of that shop and can be marketed to and managed CRM-style. Usernames are not tracked and can change at any point.

Statuses: **new** (registered, unverified), **active** (at least one verified email), **restricted** (typically non-payment; can browse but no purchases/shop management/receiving orders), **suspended** (site off-limits until resolved; account persists for username-clash checks), **deleted** (account gone).

@todos: shipping saves must have region or country before proceeding; DB integrity loses to ease of architecture here, investigate; introduce weight-based shipping.

## Inventory module: open questions

- When a product option is removed, the related SKU disappears; this dynamism may cause reporting issues.
- Should shops be able to specify their own SKU designations?
- Remove `code` from product attributes; only the attribute options need them.
- Split archive from status? A product could be archived (hidden from the seller's daily view) yet still active/buyable.
- SKU codegen note: the documented scheme is based on the generation process used during data migration; the final attribute-code spec was explicitly undecided, including how much control shops get over attributes/options.

## Fulfilment module: rationale and open questions

The inventory process supplies three item types with differing lifecycles, categorised by fulfilment as **orders, subscriptions, and invoice runs**. The point of an order is to codify the details of a request that may otherwise change (address, description, price); this is the origin of crystallisation (ADR-0004).

Entity roles: **item** (crystallised line), **group** (order; first-class citizen, holds status, ties process to shop and user, entry point for queries), **invoice** (tracks payment progress, separate from the group because it tracks a different progression), **transaction**. Delivery address and payment method entity shapes were marked "????" (unresolved).

**Why the group (order) must stay first-class** (the spec's "brief history lesson", protecting future refactors):

1. Removing the group and shifting responsibility onto items looks simpler but: (a) invoice/address/shop/user relationships get duplicated per item, n*4 rows instead of 4+n; (b) that means maintaining 4 pivot tables in tandem or related entities point at different item sets, pushing integrity into the application layer, which will get it wrong at least once; (c) items owning statuses means cancelling an order requires n status changes that must all succeed; (d) in short, making things easier to split apart makes them harder to keep together.
2. Linking invoices directly to order items: since an order cannot be altered once placed and invoices are created after, it's reasonable for the invoice to take item info from the order. Only a fully automated system allowing post-placement edits would force direct invoice↔item relations.
3. Aggregation rule (verbatim constraint): **if an invoice can be aggregated, then the order_sku to invoice/order relationship cannot be maintained without also aggregating the order.**

Open questions from the spec: replicate line items from order/subscription to invoice, or link? Crystallise product/plan links (as then) or reference, and what happens to crystallised copies when a plan updates? Should invoices use an external reference as PK given accounting demands the reference never change?

Order lifecycle details not carried into the build: returns/exchange reversals are conditional on time between dispatch and return, item condition, and the shop's returns policy. Order merging was rejected: merges with separate delivery addresses need manual per-item selection; integrity demands merge-as-cancellations-plus-reorder; address changes after dispatch risk mis-delivery; and it ignores buyer intent (two orders may be deliberate). `merged` status possibly needs a `related_order_id` column.

## Invoice runs and the billing process (out of scope in this build)

A product with billing_type **deferred/periodic** and billing_period **payment, weekly, monthly, or annually** creates an invoice run: a post-pay process identical to the order process except the invoice isn't generated at POS, so multiple orders can gather under one invoice. Statuses: new, cancelled, completed. Lifecycle events: create fee, cancel fee, invoicing. Only internal fees used the periodic type. @todo: merge invoices?

Billing pipeline (Coriandr-to-shop): a cron collects unpaid, payment-due order SKUs into an invoice run table; a separate process creates actual invoices from it; another notifies the shop by email; payment creates a pending transaction; gateway confirmation completes the transaction and marks the order SKU paid; another process notifies receipt. Shop-to-customer billing is identical but driven by checkout progression rather than cron. Invoice runs over 6 months old are purged. **Recurring order SKUs are never marked paid and keep generating invoices until cancelled.**

Invoices: a PAID invoice == receipt. Two guises: sales and pro-forma. Grouped by shop plus billing type/period to give consistent payment terms (due dates). References auto-generated initially, shop-customisable later. Order SKUs can appear on multiple invoices (subscriptions). Transactions and invoices are one-to-one. Open questions: paying multiple invoices with one payment (which invoice counts as paid on partial cover?), merging/splitting invoices, vouchers' interaction, and what owns the paid status (transaction, invoice, or SKU?).

## Billing module: subscriptions (out of scope in this build)

A product with billing_type **recurring** and period **weekly/monthly/annually** creates a subscription: a pre-pay set of a single subscription (item) linked to a schedule (group) that creates many invoices on a cycle. The schedule knows start, cycle length, and next-cycle date. Uniquely among fulfilment processes, subscriptions *should* have many invoices.

Statuses: **active** (initial), **paused** (subscriber-triggered halt, e.g. holiday), **blocked** (system-triggered halt, e.g. failed payment), **cancelled**.

Lifecycle: creation makes subscription + schedule + initial invoice at once. Pre-billing: days before the next cycle a cron creates the next invoice, due at cycle start. Billing: on renewal day a transaction is attempted; success marks the invoice paid, failure blocks the subscription and marks the invoice overdue. Any successful payment marks the schedule active again so blocked subscriptions unblock asap. Blocked/paused stop invoice creation, with access expiring at the start of the next cycle. Cancellation forfeits remaining days. Subscription processing must be gathered **per shop** to avoid upgrades/downgrades splitting across two runs; ad-hoc charges create orphaned invoice line items gathered into invoices; pro-rata applies on plan changes.

## Payment module extras

- Billing types (immediate, recurring, periodic) x periods (none, weekly, monthly, annually) combine into: **Purchase** (immediate+none), **Subscription** (recurring+weekly|monthly|annually), **Deferred payment** (periodic+monthly). Except recurring, only one invoice per order SKU.
- Why orders and invoices are separate tables: (1) their statuses belong to different sides of the purchase (shop vs user); (2) no guaranteed linear progression between the two sides (an order can be paid then dispatched while its invoice is voided to aggregate payments; one status column would imply regression); (3) recurring billing means one order creates many invoices while one order_sku creates one order.
- Gateway tokens (Stripe) stored against the order; optionally against the user/shop account if "save payment" chosen.
- @todo: how are refunds expressed/represented?
- Dated gateway analysis, kept for flavour only: move from Paypal to Stripe for invoice payments (fees equalise ~£5.50, Stripe cheaper above £5.90, payment guaranteed); Stripe Connect would give the smoothest checkout but forces all sellers onto Stripe and was estimated to lose the 204 sellers outside Stripe-supported regions; without Connect, cross-border rules would lock the platform to UK sellers; rollout was Stripe-for-platform-billing first, optional seller adoption, reassess making Connect mandatory.

## Shop module (out of scope in this build)

Statuses: **open** (manageable, products visible/purchasable), **hidden** (manageable, products invisible but purchasable), **closed** (cannot be managed or purchased from).

Vouchers: application scope **specific** (linked products only), **all** (everything in cart), **auto** (every product automatically discounted, i.e. a sale). Discounts % or value. Valid from/to dates. Redemption limits per user. Linked to order_skus (specific) or sales_orders (all/auto) at point of sale.

Holidays: separate from shop status; start, end, recurrence(?), message.

Categories: designed to be throwaway, purely a UI classification. One category per product at a time, no category history, so historical category/sales reports are impossible. (The new design replaces categories with labels; see `docs/services/product.md`.)

## Forum module (out of scope in this build)

The old schema was tutorial-grade; changes affecting the read/write ratio were expected to force a restructure. All updates were destructive: `updated_at` existed but no record of what changed or prior values.

Reporting inappropriate content: users report topics and posts; past a threshold an admin email is sent, with further emails as the count grows. Required measures: prevent one user spamming reports, let reporters give a reason, and stop further reports re-raising email once an admin marks the content ok.

Statuses move freely from any value to any value; all admin-only except a post's removed status. Duplicate statuses (deleted vs archived vs spam) exist purely for reporting context. Boards: active, inactive (no new topics), deleted, archived. Topics: active, pinned, inactive, deleted, spam, archived; a topic holds the title and initial post content (topic->posts->first->content was collapsed to topic->content). Posts: active, deleted, spam.

## Image module (out of scope in this build)

Definitions: **asset** (original file sent to Cloudinary), **format** (size and, more importantly, aspect ratio, e.g. product main vs thumbnail), **crop** (portion of the asset used for a format), **image** (unique combination of asset + format + optional crop visible to the end user). An asset serves multiple images (profile and product); both shops and users own assets.

Formats are platform-controlled, invisible to users: banner, profile, main, featured, thumbnail (derived), mini (derived).

Creation: adding an image to an entity offers a custom crop via a draggable box locked to the format's aspect ratio; crops are revisitable through the image console. Display: construct the Cloudinary URL, custom crop applied first, then format size; no custom crop falls back to Cloudinary's "fill" method.

Noted design tension: defining an image as asset+format existed to support separate main/thumbnail crops, but once users manage their own images, an image shared between profile and product probably needs to become two entities to keep crop updates manageable.

Images and orders: if an image's only use is an order summary (product sold and not relisted, or shop closed), create a smaller copy and remove the original asset.

