# Facts

- Roadmap item 6 in `roadmap.md`. Commit `ec88263` created the project with `mix new cart --sup`. Local toolchain: Elixir 1.20, OTP 27 (`apps/cart/mix.exs` requires `~> 1.20`).
- The HTTP server is Bandit with Plug, not Phoenix. The user decided this needs no ADR.
- The service listens on port 4000 inside the container. Compose publishes it as `127.0.0.1:4000:4000`, loopback only, as `product` does with 8080 (`compose.yaml`).
- The listen port comes from `FENNEL_CART_PORT`, default 4000, read once at start. This mirrors `FENNEL_PRODUCT_ADDR` in `apps/product/README.md`.
- The health endpoint is `GET /healthz`. It sits outside `/v1`, takes no auth, and returns 200 with `{"status":"ok"}`.
- The image is built by a multi-stage Dockerfile at `apps/cart/Dockerfile`, with `apps/cart` as the build context. The runtime stage is slim Debian and runs as a non-root user. `apps/product/Dockerfile` is the model.
- The `cart` entry in `compose.yaml` has `build: ./apps/cart`, the port mapping, and no `depends_on`, no environment tokens and no database.
- Files likely involved: `apps/cart/mix.exs`, `apps/cart/lib/cart/application.ex`, a new router module under `apps/cart/lib/cart/`, `apps/cart/config/runtime.exs`, `apps/cart/Dockerfile`, `apps/cart/.dockerignore`, `apps/cart/README.md`, `compose.yaml`, and tests under `apps/cart/test/`.
- `docs/AUTHORING.md` governs comments, README prose and commit messages. Read it first.
- The acceptance test starts what it needs with `docker compose up -d --build cart` and polls `/healthz`. It must never run `docker compose down` or remove volumes, because the developer's own stack may be running.
- `checks.sh` runs the full suites of both services: `mix test` in `apps/cart`, and `go vet` and `go test` in `apps/product`. The `product` database tests skip when `FENNEL_PRODUCT_DSN` is unset (`apps/product/README.md`).
- Credo and Dialyzer are out until the user adds them.
