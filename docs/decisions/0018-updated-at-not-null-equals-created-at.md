# 0018 - `updated_at` is NOT NULL, and equals `created_at` at INSERT

Status: Accepted

## Context

An inherited convention from the original spec was that `updated_at` was NULL at the point of `INSERT`. This meant one column was communicating both when the row was last written, and whether it had been written to at all. This build only requires the former.

Should this build ever need to know the latter, `updated_at != created_at` is the same test as `updated_at IS NOT NULL`. An update that writes the row sets `updated_at` either way. The two are not entirely equivalent though. If `updated_at` and `created_at` are not identical during `INSERT` then any comparison between the two may produce a false positive.

Allowing `updated_at` to be NULL requires `COALESCE(updated_at, created_at)` when sorting or querying on "last written".

Additional query complexity today has a higher cost and a bigger impact than the future possibility of needing to ensure timestamp synchronicity during `INSERT`.

`updated_at` **does not indicate whether an entity has changed**, only that its underlying row has been written to.

## Decision

- An `updated_at` column must be `NOT NULL` and `DEFAULT now()`.
- During `INSERT`, the same timestamp instance must be used to write both `updated_at` and `created_at`.

## Consequences

- A recency read is an `ORDER BY`, no need for `COALESCE`.
- No consumer handles a NULL timestamp.
- `updated_at != created_at` is never a false positive. A row whose timestamps differ has been written to since its insert.
- `updated_at == created_at` is true at the insert. Nothing in this ADR keeps it true, because nothing here decides what writes the column afterwards.
