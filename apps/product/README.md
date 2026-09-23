# product

The catalogue service, in Go. `docs/services/product.md` holds the design, and `contracts/product.openapi.yaml` holds the REST contract. Run every command below from `apps/product/` unless a step says otherwise.

## Database

Compose runs the service's Postgres as `product-db` (ADR-0033). It listens on host port 54321, on loopback only, and the user, password and database are all `product`.

1. From the repository root, start the database with `docker compose up -d product-db`.
2. Apply the migrations with `goose -dir migrations postgres "postgres://product:product@localhost:54321/product" up`.

The migrations use pressly/goose. If `goose --version` does not print a v3 version, install it with `go install github.com/pressly/goose/v3/cmd/goose@latest`.

The migrations hold schema only. No migration inserts rows, so a migrated database starts empty.

## Testing

`go test ./...` passes without a database. The tests that need Postgres read `FENNEL_PRODUCT_DSN`, and they skip when it is unset.

To run the database tests:

1. Start the database, as above. The tests migrate their own schema, so the second step there is optional.
2. Run `FENNEL_PRODUCT_DSN=postgres://product:product@localhost:54321/product go test ./...`.

A package whose tests need Postgres follows this protocol:

- `TestMain` creates a schema with a random name, runs the migrations into it, and loads the package's fixture from `testdata/`. It drops the schema when the run ends.
- Every connection the tests use sets `search_path` to that schema, so a test run never reads or writes the tables a developer works against, and two runs never see each other's rows.
- A fixture fixes its ids and its timestamps, so a test can assert an exact page and an exact order. A test names a row by its fixture id rather than by its position.
- Test data lives in the fixture, never in a migration.
- A test that writes must not change what another test reads. Give it its own rows, or its own transaction that it rolls back.

A run that dies before `TestMain` returns leaves its schema behind. List the leftovers with `SELECT nspname FROM pg_namespace WHERE nspname LIKE '%_test_%'`, and drop each one with `DROP SCHEMA <name> CASCADE`.
