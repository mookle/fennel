#!/usr/bin/env bash
# Stop hook for the reviewer. It may not finish without a valid verdict.json.
source "$(dirname "$0")/lib.sh"
read_input

[ -n "$TASK_DIR" ] || exit 0
v="$TASK_DIR/verdict.json"

blockers_tampered && stop_block reviewer "$BLOCKERS_TAMPERED_MSG"

[ -f "$v" ] || stop_block reviewer "Write $v before stopping."

jq -e '
  (.status | IN("pass", "fail", "blocked"))
  and (.round | type == "number")
  and (.findings | type == "array")
  and all(.findings[];
        (.id | type == "string")
        and (.severity | IN("blocking", "minor"))
        and (.owner | IN("developer", "test-engineer"))
        and (.criterion | type == "string")
        and (.evidence | type == "string" and length > 0))
' "$v" >/dev/null 2>&1 \
  || stop_block reviewer "$v does not match the schema: status (pass|fail|blocked), round (number), findings[] each with id, severity (blocking|minor), owner (developer|test-engineer), criterion, evidence."

status="$(jq -r .status "$v")"
blocking="$(jq '[.findings[] | select(.severity == "blocking")] | length' "$v")"

if [ "$status" = "pass" ] && [ "$blocking" -gt 0 ]; then
  stop_block reviewer "status is pass but there are $blocking blocking findings. Either the status is fail or the findings are minor."
fi
if [ "$status" = "fail" ] && [ "$blocking" -eq 0 ]; then
  stop_block reviewer "status is fail but no finding is blocking. Minor findings alone are a pass."
fi
if [ "$status" = "blocked" ] && ! has_open_blocker; then
  stop_block reviewer "status is blocked but there is no open blocker. To raise one, $HOW_TO_BLOCK."
fi

stop_pass reviewer
