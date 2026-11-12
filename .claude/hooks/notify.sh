#!/usr/bin/env bash
# Tell the user they are needed. Replace the body with Slack, ntfy, etc.
# Usage: notify.sh <title> [body]
msg="${1:-Claude Code needs you}"
body="${2:-}"
if command -v terminal-notifier >/dev/null 2>&1; then
  terminal-notifier -title "Agent team" -subtitle "$msg" -message "${body:-$msg}"
elif command -v osascript >/dev/null 2>&1; then
  # AppleScript quoting is fragile, so the body is left out here.
  osascript -e "display notification \"${msg//\"/\'}\" with title \"Agent team\""
elif command -v notify-send >/dev/null 2>&1; then
  notify-send "Agent team: $msg" "$body" 2>/dev/null
else
  { printf '\a' >/dev/tty; } 2>/dev/null || true
fi
exit 0
