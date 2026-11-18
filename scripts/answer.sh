#!/usr/bin/env bash
# Run by the user, never by an agent. Answers an open blocker on the current
# task by writing .tasks/<task>/resolutions/D-NNN.md.
#
# Usage: scripts/answer.sh [D-NNN] [answer...]
#
# With no id, it answers the only open blocker. It prints the blocker, then
# takes the answer from the arguments, or from stdin when it is piped, or from
# $EDITOR when run in a terminal. Inside a Claude Code session, run it with
# `!` and pass the answer as an argument: `! scripts/answer.sh "Option a"`.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

die() { echo "$1" >&2; exit 1; }

task="$(tr -d '[:space:]' < .tasks/current 2>/dev/null || true)"
[ -n "$task" ] || die "No current task: .tasks/current is missing."
dir=".tasks/$task"
[ -d "$dir" ] || die "No such task: $dir"

open=()
for f in "$dir"/blockers/D-*.md; do
  [ -f "$f" ] || continue
  b="$(basename "$f" .md)"
  [ -s "$dir/resolutions/$b.md" ] || open+=("$b")
done

if [[ "${1:-}" =~ ^D-[0-9]{3}$ ]]; then
  id="$1"; shift
  [ -f "$dir/blockers/$id.md" ] || die "No blocker $id on $task."
  [ -s "$dir/resolutions/$id.md" ] && die "$id is already answered in $dir/resolutions/$id.md."
else
  case "${#open[@]}" in
    0) die "No open blockers on $task." ;;
    1) id="${open[0]}" ;;
    *) die "Several open blockers on $task: ${open[*]}. Name one: scripts/answer.sh <id> [answer]" ;;
  esac
fi

blocker="$dir/blockers/$id.md"
cat "$blocker"
echo

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

if [ "$#" -gt 0 ]; then
  printf '%s\n' "$*" > "$tmp"
elif [ ! -t 0 ]; then
  cat > "$tmp"
else
  { echo
    echo "# Write your answer to $id above. Lines starting with # are removed."
    sed 's/^/# /' "$blocker"
  } > "$tmp"
  "${EDITOR:-vi}" "$tmp"
  grep -v '^#' "$tmp" > "$tmp.answer" || true
  mv "$tmp.answer" "$tmp"
fi

grep -q '[^[:space:]]' "$tmp" \
  || die "The answer is empty, so $id is still open. Pass it as an argument: scripts/answer.sh $id \"<answer>\""

mkdir -p "$dir/resolutions"
cp "$tmp" "$dir/resolutions/$id.md"
echo "Answered $id in $dir/resolutions/$id.md. Tell the delivery lead to resume."
