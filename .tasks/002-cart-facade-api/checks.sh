#!/usr/bin/env bash
# Every command that must exit 0 for this task to be done, one per line.
# The developer's stop hook and the delivery lead's stop hook run this file.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

(cd apps/cart && mix deps.get)
(cd apps/cart && mix format --check-formatted)
(cd apps/cart && mix compile --warnings-as-errors)
(cd apps/cart && mix test)
(cd apps/product && go vet ./...)
(cd apps/product && go test ./...)
docker compose build cart
docker run --rm -v "$PWD/contracts:/spec:ro" redocly/cli lint --extends=minimal /spec/cart.openapi.yaml
tests/acceptance/run.sh
