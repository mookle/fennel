#!/usr/bin/env bash
# Stop hook for the technical writer. It may not finish if anything it left
# changed lies outside the documentation paths, or inside a protected one.
# The guard already refuses those writes through the file tools; this catches
# the shell. It relies on the delivery lead committing each stage, so every
# uncommitted change belongs to this stage.
source "$(dirname "$0")/lib.sh"
read_input

[ -n "$TASK_DIR" ] || exit 0
blockers_tampered && stop_block technical-writer "$BLOCKERS_TAMPERED_MSG"

allowed() {
  local p="${1%/}"
  in_dir "$p" "$TASK_DIR/blockers" && return 0
  [ "$p" = "$TASK_DIR/state.json" ] && return 0
  if [ "${#DOC_PATH_DENY[@]}" -gt 0 ] && in_any "$p" "${DOC_PATH_DENY[@]}"; then return 1; fi
  in_any "$p" "${DOC_PATHS[@]}"
}

bad=""
rename_source=0
while IFS= read -r -d '' entry; do
  if [ "$rename_source" = 1 ]; then
    # The second field of a rename or copy: the original path, with no status.
    rename_source=0
    p="$entry"
  else
    case "${entry:0:1}" in R|C) rename_source=1 ;; esac
    p="${entry:3}"
  fi
  allowed "$p" || bad="$bad
  $p"
done < <(git status --porcelain -z --untracked-files=all 2>/dev/null)

if [ -n "$bad" ]; then
  protected=""
  [ "${#DOC_PATH_DENY[@]}" -gt 0 ] && protected=" Protected: ${DOC_PATH_DENY[*]}."
  stop_block technical-writer "You changed files you may not write. You write only: ${DOC_PATHS[*]}.$protected Undo these changes (git checkout -- <file>, or delete a file you added):$bad"
fi

stop_pass technical-writer
