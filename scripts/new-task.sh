#!/usr/bin/env bash
# Create a task folder from the templates and make it the current task.
# Usage: scripts/new-task.sh <slug>
set -euo pipefail
cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

slug="${1:?usage: scripts/new-task.sh <slug>}"
mkdir -p .tasks
# An approved task leaves the main tree for its own worktree, so number from
# the task branches and the base branch as well as the drafts here.
BASE_BRANCH="main"
# shellcheck disable=SC1091
[ -f .claude/team.conf ] && source .claude/team.conf
last="$( { ls .tasks 2>/dev/null
           git for-each-ref --format='%(refname:lstrip=3)' refs/heads/task/ 2>/dev/null
           git ls-tree --name-only "$BASE_BRANCH:.tasks/" 2>/dev/null
         } | grep -Eo '^[0-9]+' | sort -n | tail -n 1 || true)"
num="$(printf '%03d' $(( 10#${last:-0} + 1 )))"
name="$num-$slug"
dir=".tasks/$name"

mkdir -p "$dir"
cp .claude/templates/{done.md,facts.md,checks.sh,state.json} "$dir/"
chmod +x "$dir/checks.sh"
echo "$name" > .tasks/current

echo "Created $dir and set it as the current task."
