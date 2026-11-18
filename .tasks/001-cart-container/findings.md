## Round 1
### F-1 · blocking · owner: developer
Criterion: Outcome 2, 5, 6 and 7
Evidence: No test exercises the running container. `tests/acceptance/` holds only `run.sh` and `README.md`, with no `001-cart-container.test.sh`, so `run.sh` runs zero files and passes vacuously. facts.md states that the acceptance test starts the service with `docker compose up -d --build cart` and polls `/healthz`; it does not exist. `apps/cart/test/cart/router_test.exs` calls the router in-process and covers Outcome 3 and 4 only, and `checks.sh` line `docker compose build cart` covers Outcome 1. Nothing would fail if `/healthz` did not answer within 60 seconds in the container, if the port were published on 0.0.0.0, if the image ran as root, or if `cart` gained a `depends_on`. By hand, all four currently hold: 200 after 1 s, `docker compose port cart 4000` prints `127.0.0.1:4000`, `id -u` prints `10001`, and `cart` runs alone.
Reproduce: ls tests/acceptance/*.test.sh; tests/acceptance/run.sh  (prints "0 passed, 0 failed")

### F-2 · minor · owner: developer
Criterion: Judged criterion 4 (README follows docs/AUTHORING.md and the naming rules)
Evidence: apps/cart/README.md:3 opens with "The Cart service". CLAUDE.md's naming rule makes capitalised Cart the container and lowercase `cart` the service, and docs/services/cart.md calls it `cart`. "The `cart` service" or a description in the style of product's "The catalogue service" avoids the mix.
Reproduce: sed -n 3p apps/cart/README.md

## Round 2
F-1 is resolved: `tests/acceptance/001-cart-container.test.sh` covers Outcome 1 to 7 against the running container, and `tests/acceptance/run.sh` reports "1 passed, 0 failed". The healthcheck added under the re-approved judged criterion (compose.yaml:50-69) reports `healthy`, and the same probe against `/nope` exits 1, so it fails on a non-200. The runtime image holds no `elixir`, `mix`, `iex` or `erl` on `PATH`, only the release's bundled ERTS. The diff stays in scope, and the `product` and `product-db` entries are unchanged.

### F-3 · minor · owner: developer
Criterion: Judged criterion 4 (README follows docs/AUTHORING.md and the naming rules)
Evidence: F-2 from round 1 still stands. apps/cart/README.md:3 still opens with "The Cart service", which uses the capitalised domain term for the service.
Reproduce: sed -n 3p apps/cart/README.md
