#!/usr/bin/env bash
# Stop hook for the test engineer. It may not finish until:
#   - there are new or changed files under the acceptance test folder,
#   - those tests currently fail (the feature does not exist yet).
source "$(dirname "$0")/lib.sh"
read_input

[ -n "$TASK_DIR" ] || exit 0
blockers_tampered && stop_block test-engineer "$BLOCKERS_TAMPERED_MSG"
has_open_blocker && stop_pass test-engineer

if [ -z "$(git status --porcelain -- "$ACCEPTANCE_DIR" 2>/dev/null)" ]; then
  stop_block test-engineer "No new or changed acceptance tests under $ACCEPTANCE_DIR/. Write one test per line of the Outcome in done.md."
fi

if [ -n "$ACCEPTANCE_CMD" ]; then
  if bash -c "$ACCEPTANCE_CMD" >/dev/null 2>&1; then
    stop_block test-engineer "The acceptance tests pass before any implementation exists, so they do not test the new behaviour. Make them fail for the right reason."
  fi
fi

stop_pass test-engineer
