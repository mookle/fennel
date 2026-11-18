#!/usr/bin/env bash
# Every command that must exit 0 for this task to be done, one per line.
# The developer's stop hook and the delivery lead's stop hook run this file.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

(cd apps/product && test -z "$(gofmt -l .)")
(cd apps/product && go vet ./...)
(cd apps/product && go test ./...)
docker compose build product
docker run --rm -v "$PWD/contracts:/spec:ro" redocly/cli lint --extends=minimal /spec/product.openapi.yaml
tests/acceptance/run.sh
