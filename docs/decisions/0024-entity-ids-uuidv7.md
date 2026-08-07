# 0024 - Entity ids are UUIDv7, minted by the owning service

Status: Accepted

## Context

The build should not expose internal identifiers via an API. The existing wire contracts type every id as an opaque string, and the examples show prefixed, ULID-flavoured values such as `prd_01HZA1`. The question is two-fold: what format should an identifier take, and at what layer should id generation occur? There are three alternatives for format:

- **UUIDv4 keys.** A v4 is random, so each insert lands at a random point in the id index instead of at the end of it. Writes spread across every page, and pages split as they fill, where a time-ordered id keeps adding to one edge. Postgres does not store the table rows in key order, so none of this costs anything in the table itself. The whole cost sits in the indexes, and every id column has one.

- **Integer primary keys with a public UUIDv4.** Every row carries two ids: an integer PK for foreign keys and joins, and a UUID for the outside world. The integer shrinks the join columns and associated indexes, which pays off in systems large enough for the byte savings to count. It saves nothing on lookups, because the public id needs a unique index of its own to be found by, and probing that index is the same lookup cost either way. However, this approach creates two identifiers on every row; the foreign keys hold one, and the payloads hold the other, and every piece of code has to know which of the two it should care about.

- **UUIDv7 keys.** A v7 holds a millisecond timestamp in its first 48 bits and random data in the rest, so the values sort by creation time and each insert lands at the end of the index, negating the insert cost a v4 carries. The 74 random bits leave it as hard to guess as a v4. Additionally, a v7 is still an ordinary UUID, so it costs nothing to adopt: the same `uuid` column, the same driver and library support, and the same `format: uuid` in the contracts. The downside is that the value discloses when its row was created, to the millisecond.

The layer is the second question, and the format does not answer it. Postgres 18 could provide a default using its native `uuidv7()` function. One database function beats two client libraries, because there are fewer dependencies, less application code to maintain, and sub-millisecond precision that the Go and Elixir libraries do not give. A database default also covers any writes that occur outside the application layer, such as raw SQL inserting test seed data.

What a database default cannot do is enforce, because any `INSERT` that names the `id` column takes precedence. Enforcement would require a trigger that overwrites what the caller sent. Reading the id back after the insert would serve every write in this build, so the database _could_ supply all of them bar one: the event envelope carries an `id` but is never a row, so application code mints that one whatever the columns default to. A test that builds a row with a known id supplies its own as well. An explicit id is therefore the "standard" path, and a default would fire only on writes where the application layer opts not to.

## Decision

- Every **entity** id is a UUIDv7, and the owning service mints it in application code.
- The application layer supplies a UUIDv7 id on every write it makes, and the database default catches direct `INSERT` queries. Application code must never rely on the default.
- No application code reads an id back from the database.
- The column type is `uuid`, with `DEFAULT uuidv7()`.
- JSON carries an id as a plain lowercase UUID string, with `format: uuid` in the contracts. There are no prefixes.
- Identifiers that represent out-of-scope entities remain opaque, with no constrained shape.

## Consequences

- Bookkeeping and join tables keep their natural keys.
- Entity identifiers disclose their creation time to millisecond precision. For this build that is accepted: `created_at` is already on the wire for catalogue entities, and an Order's placement time is not a secret from anyone who holds its id.
- Index writes are kept append-mostly, and a recency sort on an id is possible. N.B. `created_at` remains the correct column for recency data (ADR-0018).
- A row does not record which layer minted its id and nothing downstream can tell them apart. 
- Postgres 18 is the minimum version everywhere, in the cluster and on the cloud target, which both offer it (ADR-0009).
