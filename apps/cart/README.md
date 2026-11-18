# cart

The Cart service, in Elixir, with Bandit and Plug for HTTP. `docs/services/cart.md` holds the design. Run every command below from `apps/cart/` unless a step says otherwise.

## Running the service

Compose runs the service as `cart`, and it listens on host port 4000, on loopback only (ADR-0033). From the repository root, start it with `docker compose up -d --build cart`. The service depends on no other service, so it starts alone.

The image runs a `mix release`. The service reads its configuration from the environment, once, at start. A changed value takes a restart.

- `FENNEL_CART_PORT` is the listen port. It defaults to `4000`.

`GET /healthz` reports that the service is up. It takes no token:

```sh
curl -s localhost:4000/healthz
```

The response is `{"status":"ok"}`.

Compose probes the same endpoint as the `cart` healthcheck, so `docker compose ps` shows the service as healthy once it answers.

To run the service without Docker:

1. Fetch the dependencies with `mix deps.get`.
2. Start the service with `mix run --no-halt`.

## Testing

1. Fetch the dependencies with `mix deps.get`.
2. Run `mix test`.

The tests call the router directly and start no listener, so they run while the container holds port 4000.
