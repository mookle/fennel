# Task: Serve the `cart` facade API with fixed responses

Roadmap item 7. `cart` serves three routes with fixed responses, and `contracts/cart.openapi.yaml` describes them. This is a facade: it does no domain work, holds no state and validates no input.

## Outcome
1. `POST /v1/carts` returns 201 with a JSON body whose `id` is a lowercase UUIDv7 string.
2. Two calls to `POST /v1/carts` return the same `id`.
3. `POST /v1/carts/{id}/items` with the body `{"sku_id":"<uuid>","quantity":2}` returns 201 with a JSON body whose `sku_id` and `quantity` equal the values sent.
4. `POST /v1/carts/{id}/items` returns the same status and echo for an `{id}` that `POST /v1/carts` never returned.
5. `POST /v1/carts/{id}/checkout` returns 202 with a JSON body whose `purchase_id` is a lowercase UUIDv7 string.
6. Two calls to `POST /v1/carts/{id}/checkout` return the same `purchase_id`, and it differs from the `id` that `POST /v1/carts` returns.
7. Every response from the three routes has `Content-Type: application/json`.
8. Each of the three routes succeeds with no `Authorization` header.
9. `GET /healthz` still returns 200 with `{"status":"ok"}`.
10. `contracts/cart.openapi.yaml` passes `redocly lint` with the minimal ruleset.

## Executable checks
All commands in `checks.sh` in this folder must exit 0.

## Judged criteria
- `contracts/cart.openapi.yaml` declares the three operations with the status codes and body fields in the Outcome, and an example for each response that equals what the facade returns.
- The contract follows the conventions of `contracts/product.openapi.yaml`: OpenAPI 3.1.0, a `servers` entry for the Compose network (`http://cart:4000/v1`), and ids as `type: string, format: uuid`.
- The contract's `info.description` states that the service is a facade at this stage and names what is fixed.
- The fixed values live in one place in the code, not repeated across handlers.
- Names follow the naming rules in `CLAUDE.md`: Cart, CartItem and Purchase as `docs/ARCHITECTURE.md` defines them, and `purchase_id`, not `order_id`.

## Out of scope
- Input validation, 400 responses and 404 responses for unknown carts.
- Any state: the routes keep nothing between calls.
- The call to `product` on add (roadmap item 8) and Req.
- Publishing `purchase.submitted` on checkout (roadmap item 9).
- Auth on the `cart` API.
- `/healthz` in the contract.
- Docs under `docs/`, apart from a link to the new contract from `docs/services/cart.md` if the technical writer judges it useful.

## Escalate if
- The add-line echo needs fields beyond `sku_id` and `quantity` to make sense in the contract (for example `shop_id` or the cart id).
- A request body for `POST /v1/carts` seems necessary (for example `user_id`).
- `redocly lint` with the minimal ruleset flags a convention that `contracts/product.openapi.yaml` also breaks.
- The routes or response shapes in the contract disagree with `docs/services/cart.md` in a way that needs a doc change, not a contract change.
