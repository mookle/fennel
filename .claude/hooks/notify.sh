#!/usr/bin/env bash
# Tell the user they are needed, as "<NNN> <role>: <status>", with no body.
# Replace the sending part with Slack, ntfy, etc.
#
# Usage: notify.sh <role> <status>
#        notify.sh --hook <status>
# The --hook form is for a Notification hook. It takes the role from the
# payload's agent_type, and leaves the role out if the payload has none.
source "$(dirname "$0")/lib.sh"

role="${1:-}"
status="${2:-needs input}"

if [ "$role" = --hook ]; then
  role=""
  if [ ! -t 0 ]; then
    payload="$(cat)"
    role="$(jq -r '.agent_type // ""' <<<"$payload" 2>/dev/null || true)"
    # Temporary: keep each payload so a real run shows what it carries.
    if [ -n "$TASK_DIR" ] && [ -d "$TASK_DIR" ]; then
      mkdir -p "$TASK_DIR/.hooks"
      printf '%s\n' "$payload" >> "$TASK_DIR/.hooks/notification-payloads.jsonl"
    fi
  fi
fi

num="${TASK%%-*}"
who="$num${num:+${role:+ }}$role"
msg="${who:+$who: }$status"

if command -v terminal-notifier >/dev/null 2>&1; then
  terminal-notifier -title "Agent team" -message "$msg"
elif command -v osascript >/dev/null 2>&1; then
  osascript -e "display notification \"${msg//\"/\'}\" with title \"Agent team\""
elif command -v notify-send >/dev/null 2>&1; then
  notify-send "$msg" 2>/dev/null
else
  { printf '\a' >/dev/tty; } 2>/dev/null || true
fi
exit 0
