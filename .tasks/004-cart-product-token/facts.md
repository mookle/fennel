# Facts

- Roadmap item 5 in `roadmap.md`. Item 4 (`GET /v1/skus:batchGet` in `product`, task 003) comes before it, and item 8 (cart calls `product` on add) comes after it and consumes this task's config.
- Decisions the user made for this task:
  - `cart` reads the token and the URL in this task, not in item 8.
  - The variable names are `FENNEL_CART_PRODUCT_TOKEN` and `FENNEL_CART_PRODUCT_URL`, following `FENNEL_CART_PORT`. The URL is `http://product:8080/v1`, the server URL in `contracts/product.openapi.yaml`.
  - The token literal has one source: a YAML anchor in `compose.yaml`, for example on a top-level `x-` extension field. Not a `.env` file. The value is a local-only literal, like `dev-token`, for example `cart-token`.
  - Neither `cart` nor `product` depends on the other at start. Add no `depends_on` between them.
  - The release refuses to start if either variable is unset or empty, with an error that names the variable. Under `mix test` neither variable is required.
  - Tier 2: it touches service-to-service auth, and the token check needs the running stack.
- The application config keys are `:product_url` and `:product_token` under the `:cart` app. Item 8 reads them with `Application.fetch_env!/2`.
- `product` turns `FENNEL_PRODUCT_TOKEN_CART` into the caller `cart` (`apps/product/cmd/product/config.go`), and it rejects a token another caller already holds. The cart token must differ from `dev-token`.
- `config/runtime.exs` already reads `FENNEL_CART_PORT`. `config/config.exs` sets `server: config_env() != :test`, so the unit tests start no listener.
- The release runs as `/app/bin/cart start` (`apps/cart/Dockerfile`). `bin/cart rpc` evaluates an expression in the running node.
- Compose names the built image `fennel-cart`. Lines 8 to 11 run it with `docker run --rm` and `timeout`, passing the variables with `-e`, so they never start or stop a Compose service.
- The acceptance test starts what it needs with `docker compose up -d --build`, and must never run `docker compose down` or remove volumes.
- `docs/AUTHORING.md` governs comments, README prose and commit messages.
