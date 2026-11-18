# Task: Give `cart` its own token and URL for `product`

Roadmap item 5. `compose.yaml` gives `cart` a bearer token that `product` accepts, and the URL of `product` on the Compose network. `cart` reads both at start into its application config, and the release refuses to start without them. Nothing calls `product` yet (roadmap item 8).

"The resolved config" means the output of `docker compose config --format json` run from the repository root.

## Outcome
1. In the resolved config, `services.product.environment.FENNEL_PRODUCT_TOKEN_CART` is a non-empty string, and `services.cart.environment.FENNEL_CART_PRODUCT_TOKEN` equals it.
2. In the resolved config, `services.cart.environment.FENNEL_CART_PRODUCT_URL` is `http://product:8080/v1`.
3. The token's value appears exactly once in `compose.yaml`.
4. In the resolved config, neither `product` nor `cart` has a `depends_on` entry that names the other.
5. `GET /v1/products` on `product` with `Authorization: Bearer <the token from line 1>` returns 200.
6. After `docker compose up -d --wait cart`, `docker compose exec cart /app/bin/cart rpc 'IO.puts(Application.fetch_env!(:cart, :product_url))'` prints `http://product:8080/v1`.
7. After `docker compose up -d --wait cart`, `docker compose exec cart /app/bin/cart rpc 'IO.puts(Application.fetch_env!(:cart, :product_token))'` prints the token from line 1.
8. The `cart` image, run with `FENNEL_CART_PRODUCT_URL` set and `FENNEL_CART_PRODUCT_TOKEN` unset, exits non-zero within 30 seconds, and its output contains `FENNEL_CART_PRODUCT_TOKEN`.
9. The `cart` image, run with `FENNEL_CART_PRODUCT_URL` set and `FENNEL_CART_PRODUCT_TOKEN` set to the empty string, exits non-zero within 30 seconds, and its output contains `FENNEL_CART_PRODUCT_TOKEN`.
10. The `cart` image, run with `FENNEL_CART_PRODUCT_TOKEN` set and `FENNEL_CART_PRODUCT_URL` unset, exits non-zero within 30 seconds, and its output contains `FENNEL_CART_PRODUCT_URL`.
11. The `cart` image, run with `FENNEL_CART_PRODUCT_TOKEN` set and `FENNEL_CART_PRODUCT_URL` set to the empty string, exits non-zero within 30 seconds, and its output contains `FENNEL_CART_PRODUCT_URL`.
12. A token consisting of empty spaces should be treated as empty.
13. A URL consisting of empty spaces should be treated as empty.

## Executable checks
All commands in `checks.sh` in this folder must exit 0.

## Judged criteria
- `compose.yaml` holds the token literal once, under a YAML anchor, and both services refer to it through an alias. A comment says the value is for local use only, as the `dev-token` comment does.
- The comment in `compose.yaml` that says "cart takes its token when it joins the stack" no longer describes the future.
- `config/runtime.exs` reads both variables only when the release runs (`config_env() == :prod`). `mix test` passes with neither variable set.
- `apps/cart/README.md` lists both new variables alongside `FENNEL_CART_PORT`, and says the release refuses to start without them.
- The `apps/cart/README.md` line "The service depends on no other service, so it starts alone" stays true or is reworded to stay true: `cart` still starts without `product`.

## Out of scope
- Calling `product` from `cart`, an HTTP client, and Req (roadmap item 8).
- Checking the URL's shape beyond its being present and non-empty.
- A Compose healthcheck for `product`.
- Any change to `apps/product` code. `product` already accepts any `FENNEL_PRODUCT_TOKEN_*` variable as a caller.
- A root `.env` file, or interpolation from the host environment.
- Docs under `docs/`.

## Escalate if
- `docker compose config` does not resolve the anchor into both services' environment, or Compose rejects an anchor on an `environment` value.
- The release cannot refuse to start from `config/runtime.exs` with an error that names the variable, and the check needs code elsewhere, for example in `Cart.Application`.
- Reading the variables only under `:prod` leaves `Application.fetch_env!(:cart, :product_url)` or `:product_token` raising in a way that breaks `mix test` or the existing unit tests.
- Lines 8 to 11 cannot run the image without Compose starting `product` or another service, or without touching the running stack.
