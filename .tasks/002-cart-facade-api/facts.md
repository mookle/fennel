# Facts

- Roadmap item 7 in `roadmap.md`. It depends on task 001 (the container and `/healthz`). Approve this task only after 001 is on `main`.
- Routes, as decided by the user:
  - `POST /v1/carts` returns 201 and `{"id": "<fixed uuid>"}`.
  - `POST /v1/carts/{id}/items` returns 201 and echoes `sku_id` and `quantity` from the request body.
  - `POST /v1/carts/{id}/checkout` returns 202 and `{"purchase_id": "<fixed uuid>"}`.
- No validation in the facade (user decision). Behaviour for a missing or malformed body is undefined, and no test covers it.
- The cart API takes no auth for now. ADR-0033 governs service-to-service tokens, and no ADR names a caller of `cart`.
- The fixed ids are valid lowercase UUIDv7 strings (ADR-0024). Choose them once, and use the same values in the contract examples.
- Write the contract to `contracts/cart.openapi.yaml`. Model it on `contracts/product.openapi.yaml`.
- Check the contract with `docker run --rm -v "$PWD/contracts:/spec:ro" redocly/cli lint --extends=minimal /spec/cart.openapi.yaml` from the repository root. `checks.sh` runs this command.
- Bandit and Plug serve HTTP (task 001). Req is the HTTP client for later tasks, and this task adds no client.
- `docs/AUTHORING.md` governs comments, contract descriptions, README prose and commit messages.
- The acceptance test starts what it needs with `docker compose up -d --build cart` and polls `/healthz`. It must never run `docker compose down` or remove volumes.
- Credo and Dialyzer are out until the user adds them.
