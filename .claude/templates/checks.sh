#!/usr/bin/env bash
# Every command that must exit 0 for this task to be done, one per line.
# The developer's stop hook and the delivery lead's stop hook run this file.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

