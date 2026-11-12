#!/usr/bin/env bash
# Shared helpers. Sourced by every hook.

PROJECT="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
cd "$PROJECT" || exit 0

ACCEPTANCE_DIR="tests/acceptance"
SRC_DENY=("src")
PUBLIC_ALLOW=()
DOC_PATHS=("docs" "README.md" "CHANGELOG.md")
DOC_PATH_DENY=()
BASE_BRANCH="main"
ACCEPTANCE_CMD=""
MAX_STOP_BLOCKS=3
# shellcheck disable=SC1091
[ -f .claude/team.conf ] && source .claude/team.conf

TASK=""
[ -f .tasks/current ] && TASK="$(tr -d '[:space:]' < .tasks/current)"
TASK_DIR=""
[ -n "$TASK" ] && TASK_DIR=".tasks/$TASK"

# Path relative to the project root. Paths outside the project stay absolute.
rel() {
  local p="$1"
  p="${p#"$PROJECT"/}"
  p="${p#./}"
  printf '%s' "$p"
}

# in_dir <path> <dir-or-file>: true if path is that entry or inside it.
in_dir() {
  [[ "$1" == "$2" || "$1" == "$2"/* ]]
}

# in_any <path> <entries...>
in_any() {
  local p="$1"; shift
  local d
  for d in "$@"; do
    in_dir "$p" "$d" && return 0
  done
  return 1
}

# Deny a tool call. Exit 2 blocks it and shows the message to the agent.
deny() {
  printf 'BLOCKED: %s\n' "$1" >&2
  exit 2
}

sha256_check() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum -c --quiet "$1"; else shasum -a 256 -c "$1" >/dev/null; fi
}

# True if the locked files in the task folder still match their approved hashes.
lock_intact() {
  [ -n "$TASK_DIR" ] && [ -f "$TASK_DIR/LOCKED" ] || return 1
  ( cd "$TASK_DIR" && sha256_check LOCKED ) >/dev/null 2>&1
}

# Blockers: one file per question in blockers/D-NNN.md, written by any role.
# The user answers by creating resolutions/D-NNN.md. A blocker is open while
# no non-empty resolution file has the same name.

# Print the id of each open blocker, one per line.
open_blockers() {
  [ -n "$TASK_DIR" ] || return 0
  local f id
  for f in "$TASK_DIR"/blockers/D-*.md; do
    [ -f "$f" ] || continue
    id="$(basename "$f" .md)"
    [ -s "$TASK_DIR/resolutions/$id.md" ] || printf '%s\n' "$id"
  done
}

has_open_blocker() {
  [ -n "$(open_blockers)" ]
}

# Print the id of each resolution that matches no blocker (a mistyped name).
stray_resolutions() {
  [ -n "$TASK_DIR" ] || return 0
  local f id
  for f in "$TASK_DIR"/resolutions/*.md; do
    [ -f "$f" ] || continue
    id="$(basename "$f" .md)"
    [ -f "$TASK_DIR/blockers/$id.md" ] || printf '%s\n' "$id"
  done
}

# True if a committed blocker has been edited or deleted. A blocker is
# written once; the guard stops edits through the file tools, and this
# catches the shell.
blockers_tampered() {
  [ -n "$TASK_DIR" ] || return 1
  git status --porcelain -- "$TASK_DIR/blockers" 2>/dev/null | grep -qvE '^(\?\?|A )'
}
BLOCKERS_TAMPERED_MSG="A blocker file that was already committed has been changed or deleted. Blockers are written once. Restore them with: git checkout -- $TASK_DIR/blockers"

HOW_TO_BLOCK="create a blocker file in $TASK_DIR/blockers/ (format: .claude/templates/blocker.md)"

# read_input: read the hook payload from stdin. Stop hooks call this first.
HOOK_INPUT=""
read_input() {
  HOOK_INPUT="$(cat)"
}

# Send the agent back to work, up to MAX_STOP_BLOCKS times per dispatch. After
# that, let it stop and leave a .failed note the delivery lead must escalate.
# The count is keyed on the payload's agent_id, so each dispatch of a role gets
# the full allowance, however the previous dispatch ended. A resumed agent
# keeps its agent_id, so a stop on an open blocker clears the count too.
# stop_block <role> <message>
stop_block() {
  local role="$1" msg="$2" dir="$TASK_DIR/.hooks" n id prev_id prev_n
  mkdir -p "$dir"
  id="$(jq -r '.agent_id // .session_id // ""' <<<"$HOOK_INPUT" 2>/dev/null)"
  { read -r prev_id prev_n < "$dir/$role.blocks"; } 2>/dev/null || true
  if [ "${prev_id:-}" = "$id" ]; then n=$(( ${prev_n:-0} + 1 )); else n=1; fi
  echo "$id $n" > "$dir/$role.blocks"
  if [ "$n" -gt "$MAX_STOP_BLOCKS" ]; then
    printf '%s\n' "$msg" > "$dir/$role.failed"
    rm -f "$dir/$role.blocks"
    exit 0
  fi
  # Block through JSON on stdout, not exit 2. Claude Code treats an exit 2 from
  # a Stop hook whose stderr mentions a missing file as a non-blocking error,
  # and checks output can easily contain "No such file or directory".
  block "$(printf '%s\n(Attempt %s of %s. If you cannot fix this, %s and stop.)' \
    "$msg" "$n" "$MAX_STOP_BLOCKS" "$HOW_TO_BLOCK")"
}

# block <reason>: keep the agent working, with reason as the explanation.
block() {
  jq -n --arg r "$1" '{decision: "block", reason: $r}'
  exit 0
}

stop_pass() {
  rm -f "$TASK_DIR/.hooks/$1.blocks" "$TASK_DIR/.hooks/$1.failed"
  exit 0
}
