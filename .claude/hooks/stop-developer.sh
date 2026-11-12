#!/usr/bin/env bash
# Stop hook for the developer. It may not finish until:
#   - the locked files are untouched,
#   - the acceptance tests are untouched,
#   - every command in checks.sh exits 0.
# An open blocker is always a valid way to stop.
source "$(dirname "$0")/lib.sh"
read_input

[ -n "$TASK_DIR" ] || exit 0
blockers_tampered && stop_block developer "$BLOCKERS_TAMPERED_MSG"
has_open_blocker && stop_pass developer

if [ -f "$TASK_DIR/LOCKED" ] && ! lock_intact; then
  stop_block developer "done.md, facts.md or checks.sh no longer matches the approved version. Restore them with: git checkout -- $TASK_DIR"
fi

if [ -n "$(git status --porcelain -- "$ACCEPTANCE_DIR" 2>/dev/null)" ]; then
  stop_block developer "Files under $ACCEPTANCE_DIR/ have changed. Restore them with: git checkout -- $ACCEPTANCE_DIR (and remove any files you added there)."
fi

if [ -f "$TASK_DIR/checks.sh" ]; then
  mkdir -p "$TASK_DIR/.hooks"
  log="$TASK_DIR/.hooks/checks.log"
  if ! bash "$TASK_DIR/checks.sh" >"$log" 2>&1; then
    stop_block developer "checks.sh is failing. Last 30 lines:
$(tail -n 30 "$log")"
  fi
fi

stop_pass developer
