#!/usr/bin/env bash
# Stop hook for the delivery lead (the main session). It may end its turn only
# when the task is finished or a blocker is waiting for the user.
source "$(dirname "$0")/lib.sh"
read_input

# No active task: nothing to enforce.
[ -n "$TASK_DIR" ] && [ -d "$TASK_DIR" ] || exit 0

# A dispatched role is still running in the background. The task is mid-stage,
# so there is nothing to check yet: Claude Code wakes this session when the
# role finishes. settings.json turns background tasks off, so this is a
# fallback for a session where that setting did not apply.
if jq -e '[.background_tasks[]? | select(.type == "subagent")] | length > 0' <<<"$HOOK_INPUT" >/dev/null 2>&1; then
  exit 0
fi

fail() { block "$1"; }

blockers_tampered && fail "$BLOCKERS_TAMPERED_MSG"

if has_open_blocker; then
  open="$(open_blockers | paste -sd' ' -)"
  stray="$(stray_resolutions | paste -sd' ' -)"
  msg="Open blockers on $TASK: $open. Answer each by creating $TASK_DIR/resolutions/<id>.md, then tell the delivery lead to resume."
  [ -n "$stray" ] && msg="$msg Resolutions that match no blocker (check the file name): $stray."
  "$(dirname "$0")/notify.sh" "Blocker on $TASK: $open" || true
  # Shown to the user, not to the agent.
  jq -n --arg m "$msg" '{systemMessage: $m}'
  exit 0
fi

# Tell the user something and let the session stop. Not shown to the agent.
tell_user() {
  "$(dirname "$0")/notify.sh" "$1" || true
  jq -n --arg m "$1" '{systemMessage: $m}'
  exit 0
}

[ -f "$TASK_DIR/LOCKED" ] \
  || tell_user "Task $TASK is not approved. Run: scripts/approve.sh $TASK"
[ "$(git branch --show-current 2>/dev/null)" = "task/$TASK" ] \
  || tell_user "Task $TASK runs in its own worktree, on branch task/$TASK. Start the delivery lead there."
lock_intact \
  || fail "The locked task files have changed since approval. Restore them (git checkout -- $TASK_DIR) and raise a blocker."

ls "$TASK_DIR"/.hooks/*.failed >/dev/null 2>&1 \
  && fail "A role hit its retry cap ($(ls "$TASK_DIR"/.hooks/*.failed | xargs -n1 basename)). Raise a blocker with the failure and stop."

tier="$(jq -r '.tier // 2' "$TASK_DIR/state.json" 2>/dev/null || echo 2)"

if [ "$tier" -ge 1 ]; then
  [ "$(jq -r '.status // ""' "$TASK_DIR/verdict.json" 2>/dev/null)" = "pass" ] \
    || fail "Not done: the reviewer has not passed this task. Follow the routing table, or raise a blocker if you are stuck."
fi

if [ -f "$TASK_DIR/checks.sh" ] && ! bash "$TASK_DIR/checks.sh" >/dev/null 2>&1; then
  fail "Not done: checks.sh is failing. Dispatch the developer."
fi

[ "$(jq -r '.stage // ""' "$TASK_DIR/state.json" 2>/dev/null)" = "ready" ] \
  || fail "Checks and review have passed. Finish the task: set stage to ready in state.json, then commit everything with the review summary as the commit message (see Finishing the task)."
[ -z "$(git status --porcelain 2>/dev/null)" ] \
  || fail "There are uncommitted changes. Commit everything with the review summary as the commit message, then stop."

# The final commit message is the review summary. Send it to the user.
summary="$(git log -1 --format=%B)"
"$(dirname "$0")/notify.sh" "Ready for review: $TASK" "$summary" || true
jq -n --arg m "Task $TASK is ready for review on branch task/$TASK.

$summary" '{systemMessage: $m}'
exit 0
