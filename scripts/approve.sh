#!/usr/bin/env bash
# Run by the user, never by an agent. Approves a task definition.
#
# First approval, run in the main tree:
#   1. records hashes of done.md, facts.md and checks.sh in LOCKED,
#   2. creates a worktree named after the task folder, on branch task/<name>,
#      cut from the base branch,
#   3. moves the task folder into the worktree and commits it there.
# The pipeline then runs in that worktree.
#
# Re-approval, run inside the task's worktree after you change the
# definition: records the new hashes and commits them.
#
# Usage: scripts/approve.sh [task-name]   (defaults to .tasks/current)
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

die() { echo "$1" >&2; exit 1; }

BASE_BRANCH="main"
WORKTREE_ROOT=""
# shellcheck disable=SC1091
[ -f .claude/team.conf ] && source .claude/team.conf
[ -n "$WORKTREE_ROOT" ] || WORKTREE_ROOT="../$(basename "$PWD")-worktrees"

name="${1:-$(tr -d '[:space:]' < .tasks/current 2>/dev/null || true)}"
[ -n "$name" ] || die "No task named, and .tasks/current is missing."
[[ "$name" =~ ^[A-Za-z0-9._-]+$ ]] || die "Not a task folder name: $name"
dir=".tasks/$name"
[ -d "$dir" ] || die "No such task: $dir"
branch="task/$name"

if command -v sha256sum >/dev/null 2>&1; then sum="sha256sum"; else sum="shasum -a 256"; fi
lock() { ( cd "$1" && $sum done.md facts.md checks.sh > LOCKED ); }

# Re-approval inside the task's worktree.
if [ "$(git branch --show-current)" = "$branch" ]; then
  lock "$dir"
  echo "$name" > .tasks/current
  git add "$dir" .tasks/current
  git commit -q -m "Approve task $name" || true
  echo "Locked $dir again. Tell the delivery lead to resume."
  exit 0
fi

# First approval, from the main tree.
git rev-parse -q --verify "refs/heads/$BASE_BRANCH" >/dev/null \
  || die "Base branch '$BASE_BRANCH' does not exist. Set BASE_BRANCH in .claude/team.conf."
git cat-file -e "$BASE_BRANCH:.claude/hooks/guard.sh" 2>/dev/null \
  || die "The agent team files are not on '$BASE_BRANCH'. Commit them there first: the worktree is cut from '$BASE_BRANCH' and needs the hooks."
git rev-parse -q --verify "refs/heads/$branch" >/dev/null \
  && die "Branch $branch already exists. To re-approve, run this script inside the task's worktree."
wt="$WORKTREE_ROOT/$name"
[ -e "$wt" ] && die "$wt already exists."

lock "$dir"
mkdir -p "$WORKTREE_ROOT"
git worktree add -q -b "$branch" "$wt" "$BASE_BRANCH"
wt="$(cd "$wt" && pwd)"

# Copy the definition across, without the hooks' scratch folder.
mkdir -p "$wt/$dir"
for f in "$dir"/*; do
  cp -R "$f" "$wt/$dir/"
done
echo "$name" > "$wt/.tasks/current"
# Local, untracked permission rules do not follow a checkout. Copy them over.
[ -f .claude/settings.local.json ] && cp .claude/settings.local.json "$wt/.claude/settings.local.json"

git -C "$wt" add "$dir" .tasks/current
git -C "$wt" commit -q -m "Approve task $name"

# The task now lives on its branch. Remove the draft from the main tree.
rm -rf "${dir:?}"
if [ "$(tr -d '[:space:]' < .tasks/current 2>/dev/null || true)" = "$name" ]; then
  if git ls-files --error-unmatch .tasks/current >/dev/null 2>&1; then
    git checkout -q -- .tasks/current
  else
    rm -f .tasks/current
  fi
fi

cat <<MSG
Approved $name.
  Worktree: $wt
  Branch:   $branch (cut from $BASE_BRANCH)

Start the pipeline with:
  cd "$wt" && claude --agent delivery-lead

Accept the trust prompt for the new folder, or the role hooks will not run.
When the task is merged, remove the worktree with: git worktree remove "$wt"
MSG
