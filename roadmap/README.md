# Roadmap

The priority-ordered index of work remaining. An item links to a document in `tasks/` when it needs more detail than one line.

## Objective: the demo flow runs end to end

`docker compose up` starts the stack, and one script walks the flow:

1. List the catalogue.
2. Add SKUs from two shops to a cart.
3. Submit the Purchase.
4. Show one accepted Order per shop.
5. Show the stock decremented.

The deadline is 2026-11-04. Wire every service with fixed data first, then put the test and CI around it, then replace each facade with the real service one at a time. The acceptance policy in `order` accepts at once (`contracts/events.md`), so the flow has no accept step. Day 7 is slack for whatever slips.

### Phase 0: reset

1. Relax the prime directives that put an ADR before code.
2. Break-down CLAUDE.md into more atomic chunks.

### Phase 1: fixed data facades (day 1)

Each facade serves or publishes fixed data, but the wiring between the services is real: Compose networking, the bearer tokens and RabbitMQ. A facade does no domain work.

3. Add RabbitMQ to `compose.yaml`, with a durable quorum queue for each event.
4. Make `/v1/skus:batchGet` a GET in the contract. Carry the ids as a repeated `sku_ids` query parameter with at most 100 values, return a 400 above that, and keep the response shape. Update the references in `docs/services/cart.md` and `docs/services/product.md`. Then serve it in `product` with the contract's `SkuBatch` example.
5. Give `cart` its own token for `product` in `compose.yaml`.
6. Create the Mix project under `apps/cart`, with a Dockerfile and an entry in `compose.yaml`.
7. Serve a minimal HTTP API in `cart` with fixed responses: create a cart, add a line, and check out.
8. On add, call `GET /v1/skus:batchGet` on `product`, so the call proves the token and the network.
9. On checkout, publish a fixed `purchase.submitted`.
10. Create the Mix project under `apps/order`, with its own Postgres, a Dockerfile and an entry in `compose.yaml`.
11. Consume `purchase.submitted` in `order`, log it, and publish a fixed `order.accepted`.
12. Consume `order.accepted` in `product`, log it, and ack it.

### Phase 2: script and CI cradle (day 2)

13. Write the demo script that walks the flow against the stack. Read the Orders through `psql` until `order` serves a read.
14. Add an end-to-end test that asserts the target behaviour of the flow. Mark each check a facade cannot satisfy as pending.
15. Add a GitHub Actions workflow that runs each service's tests and the end-to-end test against the Compose stack.

### Phase 3: service build-out, one at a time (days 3 to 5)

Replace one facade at a time, and keep CI green throughout. When a build-out makes a pending check pass, remove its marker.

`product` reads:

16. Add a minimal SKU schema: `skus`, `sku_codes` and `sku_stock`. Defer the attributes and the options.
17. Seed two shops, each with one product and two SKUs.
18. Serve `GET /v1/skus:batchGet` from the database, with the ids it cannot find in `missing`.

`cart`:

19. Hold each Cart in its own process under a `Registry` and a `DynamicSupervisor` (ADR-0012). Defer the snapshots and the Postgres database.
20. Resolve an SKU through `product` on add, and store the `CartItem` with its `shop_id`.
21. On checkout, resolve the lines, check the stock, generate a `purchase_id`, and use a zero shipping cost in place of the quote.
22. Publish the real `purchase.submitted`, gate the response on the publisher confirm, and stop the cart process.

`order`:

23. Dedupe `purchase.submitted` on `purchase_id` through `processed_events`.
24. In one transaction, split the Purchase into one Order per shop, with its `OrderSku` snapshots and a `placed` row in `OrderStatusHistory`.
25. Accept each Order, and publish one `order.accepted` per Order.

`product` stock:

26. Dedupe `order.accepted` on `order_id`, and decrement `sku_stock` in the same transaction.

### Phase 4: documentation, clean-up and polish (day 6)

27. Remove what the facades left behind: fixed payloads, logging stubs and pending markers.
28. Rewrite the README for a hiring manager: what Fennel is, one diagram, the one-command run, a recording of the demo, the three files to read first, links to the strongest ADRs, and a plain account of what the agent wrote and what I wrote.
29. Put the CI badge in the README.

## After the deadline

30. Serve `GET /v1/products/{id}` with its SKUs, so the demo reaches an SKU through the catalogue and not through seeded ids.
31. Add the attributes and the options to the SKU schema.
32. Give `cart` its Postgres database. Persist the Purchase at checkout, and flush Cart snapshots on idle timeout and on drain.
33. Replace the zero shipping cost with `POST /v1/shipping/quote`.
34. Serve an order read, so the demo script needs no `psql`.
35. Decide what maintains `products.base_price` once SKUs exist (ADR-0027).
