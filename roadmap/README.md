# Roadmap

The priority-ordered index of work remaining. An item links to a document in `tasks/` when it needs more detail than one line.

## Objective 1: a buyer can add an SKU to a cart

A front end can list the catalogue, open a product, choose an SKU and hand its `sku_id` to `cart`. A Cart line holds an `sku_id`, not a product id, so the catalogue list alone is not enough: the buyer reaches an SKU through the product detail, and `cart` resolves it through the SKU endpoints. Nothing sits between the product reads and the add-to-cart call, and `cart` needs none of this running, because its test doubles come from the contract examples (ADR-0015).

4. Add the SKU schema: `attributes`, the attribute options, `skus`, `sku_options`, `sku_codes` and `sku_stock`. Extend the test fixture to match.
5. Decide what maintains `products.base_price` once SKUs exist (ADR-0027).
6. Serve `GET /v1/products/{id}`, which returns a `ProductDetail` with its attributes, options and SKUs.
7. Serve `GET /v1/skus/{id}` and `POST /v1/skus:batchGet`, which return a `ResolvedSku`.


## Done

- `catalogue.Store` reads one page of the catalogue and its total from Postgres.
- Filter products where the description or name contains `q`. Case insensitive. Escape `%` and `_`.
- Serve `GET /v1/products`. The handler parses the query into a `Filter`, calls `catalogue.Store`, and maps the page to a wire type that owns the contract's field names and writes an empty list as `[]`.
- Add the service entry point under `cmd/product`: configuration, the database pool, the HTTP server, and the bearer-token middleware (ADR-0033). Add `product` to `compose.yaml`.
