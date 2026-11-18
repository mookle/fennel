## Round 1
No findings.

Coverage: each Outcome line 1-11 maps to a check in tests/acceptance/003-product-batchget-facade.test.sh (Outcome 1-11 sections), backed by unit tests in apps/product/internal/httpapi/skus_test.go (example body, status table incl. 100/101 distinct and repeated, empty value, POST 405).

Judged criteria:
- Contract shape: contracts/product.openapi.yaml:166-197 is a `get` with no request body, one `sku_ids` query parameter (required, form, explode, uuid array, minItems 1, maxItems 100), a 400 using `BadRequest`, and the 200 schema and SkuBatch example unchanged. Pass.
- Description: contracts/product.openapi.yaml:170-173 states `missing` lists requested ids that do not exist and the three 400 cases. Pass.
- Handler: the fixed batch lives only in newBatchGetSkus (apps/product/internal/httpapi/skus.go:27-53); wire types skuBatchJSON/resolvedSkuJSON own the JSON names (skus.go:114-138); the 400 uses writeError with codeInvalidRequest (skus.go:70). Pass.
- Lint fix commit: 7df0d42 changes only the two lines (quoted SkuWrite.code description, region_id as `type: [string, "null"]`), separate from 851dee0. Pass.
- Docs: docs/services/cart.md:46 and docs/services/product.md:76,135 still say POST. Not a finding at this stage: the technical writer runs after the reviewer in the tier 2 pipeline and facts.md assigns these edits to that role. This criterion still needs checking after that stage.

Probes against the running service (all as expected): `sku_ids=` alone 400, a valid id plus an empty value 400, only an unrelated parameter 400, a comma-separated value 400, PUT 405 with `Allow: GET, HEAD`, a percent-encoded `%3A` path 200.

Scope: changes are limited to the contract, apps/product/internal/httpapi, the acceptance test and task files. No tests weakened or skipped.
