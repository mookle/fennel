#!/usr/bin/env bash
# Stand-in acceptance test runner. It runs every *.test.sh file in this folder
# and fails if any of them fails. Replace it when the project has a real
# acceptance suite; .claude/team.conf names the command (ACCEPTANCE_CMD).
set -uo pipefail
cd "$(dirname "$0")"

pass=0; failed=0
for t in ./*.test.sh; do
  [ -f "$t" ] || continue
  if bash "$t"; then
    pass=$((pass + 1)); echo "PASS ${t#./}"
  else
    failed=$((failed + 1)); echo "FAIL ${t#./}"
  fi
done

echo "$pass passed, $failed failed"
[ "$failed" -eq 0 ]
