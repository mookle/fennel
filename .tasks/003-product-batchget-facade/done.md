# Task: Serve `GET /v1/skus:batchGet` in `product` with fixed data

Roadmap item 4. The contract changes `/v1/skus:batchGet` from a POST with a JSON body to a GET that carries the ids as a repeated `sku_ids` query parameter, with at most 100 values. `product` serves it as a facade: it validates the ids, then returns the contract's `SkuBatch` example whatever ids it was given.

In every line below, "with the token" means the request carries `Authorization: Bearer dev-token`, the `dev` caller's token in `compose.yaml`, and `<uuid>` stands for a lowercase hyphenated UUID.

## Outcome
1. `GET /v1/skus:batchGet?sku_ids=<uuid>` with the token returns 200, `Content-Type: application/json`, and a body that is JSON-equal to the `value` of the `SkuBatch` example in `contracts/product.openapi.yaml`.
2. `GET /v1/skus:batchGet` with 100 distinct `sku_ids` values, none of them an id in the `SkuBatch` example, with the token returns 200 and the same body as line 1.
3. `GET /v1/skus:batchGet` with 101 distinct `sku_ids` values, with the token returns 400.
4. `GET /v1/skus:batchGet` with 101 `sku_ids` values that are all the same id, with the token returns 400.
5. `GET /v1/skus:batchGet` with no `sku_ids` parameter, with the token returns 400.
6. `GET /v1/skus:batchGet?sku_ids=<uuid>&sku_ids=not-a-uuid` with the token returns 400.
7. `GET /v1/skus:batchGet?sku_ids=<uuid>` with no `Authorization` header returns 401 with a `WWW-Authenticate: Bearer` header.
8. `GET /v1/skus:batchGet?sku_ids=<uuid>` with a bearer token that `compose.yaml` does not configure returns 401.
9. `POST /v1/skus:batchGet` with the token returns 405.
10. `GET /v1/products` with the token still returns 200.
11. `redocly lint --extends=minimal` on `contracts/product.openapi.yaml` reports no errors.

## Executable checks
All commands in `checks.sh` in this folder must exit 0.

## Judged criteria
- `contracts/product.openapi.yaml` declares `/skus:batchGet` as a `get` with no request body and one query parameter, `sku_ids`: required, `style: form`, `explode: true`, an array of `type: string, format: uuid` with `minItems: 1` and `maxItems: 100`. The operation lists a 400 that uses the existing `BadRequest` response, and its 200 schema and `SkuBatch` example are unchanged.
- The operation's description says that `missing` lists the requested ids that do not exist, and that a request with more than 100 values, no values, or a value that is not a UUID returns 400.
- The fixed response lives in one place in `apps/product`, and the handler follows the existing `httpapi` pattern: the wire types own the JSON field names, and the 400 uses `writeError` with `codeInvalidRequest`.
- The two existing lint errors are fixed in a separate commit from the batchGet change, and that commit changes only those two lines' meaning: `SkuWrite.code`'s description reads in full as one string, and `region_id` stays nullable in the OpenAPI 3.1 form.
- `docs/services/cart.md` and `docs/services/product.md` refer to `GET /v1/skus:batchGet`, and no live document still names it as a POST.

## Out of scope
- Reading SKUs from the database, or computing `missing` from the ids sent (roadmap item 18).
- The body of a 400 response. Only the status is checked.
- How the facade treats UUIDs in a form other than lowercase hyphenated (uppercase, braces, `urn:uuid:` prefix).
- A comma-separated `sku_ids` value. Only the repeated form is defined.
- The 13 lint warnings in the product contract, and every other part of the contract apart from the two lint errors.
- A token for `cart` in `compose.yaml` (roadmap item 5), and any change to `cart`.
- `GET /v1/skus/{id}`.

## Escalate if
- Go's `http.ServeMux` cannot route the literal path `/v1/skus:batchGet`, or a 405 needs code beyond registering the GET route.
- The `SkuBatch` example cannot be reproduced exactly with the existing wire types or `money` types, for example because `description` is absent on one item and the encoder emits it.
- Fixing either lint error needs a change to what the contract means, not only to how it is written.
- `redocly lint` reports a new error that the batchGet change introduces and that the shape agreed above cannot avoid.
- A document outside `docs/services/cart.md` and `docs/services/product.md` names the POST form in a way that needs more than a word change.
