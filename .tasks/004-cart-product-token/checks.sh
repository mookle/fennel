#!/usr/bin/env bash
# Every command that must exit 0 for this task to be done, one per line.
# The developer's stop hook and the delivery lead's stop hook run this file.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

docker compose config --quiet
(cd apps/cart && mix deps.get)
(cd apps/cart && mix format --check-formatted)
(cd apps/cart && mix compile --warnings-as-errors)
(cd apps/cart && env -u FENNEL_CART_PRODUCT_TOKEN -u FENNEL_CART_PRODUCT_URL mix test)
(cd apps/product && test -z "$(gofmt -l .)")
(cd apps/product && go vet ./...)
(cd apps/product && go test ./...)
docker compose build cart product
tests/acceptance/run.sh
