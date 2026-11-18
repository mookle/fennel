# Task: Package `cart` as a container and run it in Compose

Roadmap item 6. The Mix project exists already. This task adds an HTTP server with a health endpoint, a Dockerfile and a `cart` entry in `compose.yaml`.

## Outcome
1. `docker compose build cart` exits 0 from the repository root.
2. After `docker compose up -d --build cart`, `GET http://127.0.0.1:4000/healthz` returns 200 within 60 seconds.
3. The `/healthz` response has `Content-Type: application/json` and the body `{"status":"ok"}`.
4. `/healthz` returns 200 to a request with no `Authorization` header.
5. `docker compose port cart 4000` prints `127.0.0.1:4000`.
6. `docker compose exec cart id -u` prints a user id other than `0`.
7. `docker compose up -d cart` succeeds with no other service running, because `cart` depends on no other service.

## Executable checks
All commands in `checks.sh` in this folder must exit 0.

## Judged criteria
- The Dockerfile follows the shape of `apps/product/Dockerfile`: a build stage and a runtime stage, the dependency fetch in its own layer before the source copy, and comments that explain each non-obvious line.
- The image runs a `mix release`, not `mix run`, and the runtime stage holds no Elixir or Mix toolchain.
- The listen port is read from the environment at runtime (`config/runtime.exs`), not fixed at compile time.
- `apps/cart/README.md` replaces the Mix boilerplate with how to run and test the service, in the structure of `apps/product/README.md`, and follows `docs/AUTHORING.md`.
- The `Cart.hello/0` placeholder and its test are gone.
- A Compose `healthcheck` for `cart`.

## Out of scope
- Any `/v1` route. Task 002 adds them.
- `contracts/cart.openapi.yaml`. Task 002 writes it.
- `cart`'s token for `product` (roadmap item 5), and Req.
- A Postgres database for `cart`, and RabbitMQ.
- Credo and Dialyzer.

## Escalate if
- No published Elixir 1.20 builder image matches a slim Debian runtime with a compatible OTP and glibc.
- The task needs a change to the `product` or `product-db` entries in `compose.yaml`.
- Host port 4000 conflicts with anything already in `compose.yaml`.
- A dependency beyond Bandit and Plug (and their own transitive dependencies) looks necessary.
