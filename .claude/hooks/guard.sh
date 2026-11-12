#!/usr/bin/env bash
# PreToolUse guard. Usage: guard.sh <role>
# Enforces who may write (and, for the test engineer, read) which paths.
# Roles: any, product-owner, delivery-lead, test-engineer, developer,
#        reviewer, technical-writer
source "$(dirname "$0")/lib.sh"

role="${1:-any}"
input="$(cat)"
tool="$(jq -r '.tool_name // ""' <<<"$input")"
raw="$(jq -r '.tool_input.file_path // .tool_input.notebook_path // .tool_input.path // ""' <<<"$input")"
pattern="$(jq -r '.tool_input.pattern // ""' <<<"$input")"
# A main session's frontmatter hooks also fire for its subagents' tool calls,
# with agent_type naming the subagent. Each role's rules apply only to that
# role; the subagent's own frontmatter hook guards it.
agent="$(jq -r '.agent_type // ""' <<<"$input")"
[ "$role" != any ] && [ -n "$agent" ] && [ "$agent" != "$role" ] && exit 0
path="$(rel "$raw")"

is_task_file() { [[ "$path" == .tasks/*/"$1" ]]; }

case "$tool" in
  Edit|Write|MultiEdit|NotebookEdit)
    # 1. The lock applies to every role.
    is_task_file LOCKED && deny "LOCKED is created only by scripts/approve.sh, run by the user."
    if [ -n "$TASK_DIR" ] && [ -f "$TASK_DIR/LOCKED" ]; then
      for f in done.md facts.md checks.sh; do
        [ "$path" = "$TASK_DIR/$f" ] && deny "$f is locked. To change it, $HOW_TO_BLOCK and return blocked."
      done
    fi
    # 2. No agent rewrites the team's own rules.
    in_dir "$path" ".claude" && deny "Agents do not edit .claude/. Raise a blocker instead."
    # 3. Only the user answers a blocker.
    [[ "$path" == .tasks/*/resolutions/* ]] && deny "Only the user writes resolutions. Stop and wait for the answer."
    # 4. Any role may raise a blocker, as a new file. An existing one is never changed.
    if [[ "$path" == .tasks/*/blockers/* ]]; then
      [[ "$(basename "$path")" =~ ^D-[0-9]{3}\.md$ ]] || deny "Name a blocker D-NNN.md, using the next free number in the folder."
      [ -e "$path" ] && deny "$(basename "$path") already exists. Blockers are written once: raise a new one with the next free number."
      exit 0
    fi

    # 5. Per-role write rules.
    case "$role" in
      any) ;;
      product-owner)
        in_dir "$path" ".tasks" || deny "The product owner writes only inside .tasks/."
        ;;
      delivery-lead)
        is_task_file state.json || [ "$path" = ".tasks/current" ] \
          || deny "The delivery lead writes only state.json. Dispatch the role that owns this file."
        ;;
      test-engineer)
        in_dir "$path" "$ACCEPTANCE_DIR" || deny "The test engineer writes only inside $ACCEPTANCE_DIR/."
        ;;
      developer)
        in_dir "$path" "$ACCEPTANCE_DIR" && deny "Acceptance tests are read-only. If one contradicts done.md, raise a blocker and return blocked."
        in_dir "$path" ".tasks" && deny "The developer does not write task files other than a new blocker."
        [[ "$path" == /* ]] && deny "Writes outside the project are not allowed."
        ;;
      reviewer)
        is_task_file findings.md || is_task_file verdict.json \
          || deny "The reviewer writes only findings.md and verdict.json. Report faults; do not fix them."
        ;;
      technical-writer)
        if [ "${#DOC_PATH_DENY[@]}" -gt 0 ] && in_any "$path" "${DOC_PATH_DENY[@]}"; then
          deny "The technical writer may not write: ${DOC_PATH_DENY[*]}."
        fi
        in_any "$path" "${DOC_PATHS[@]}" || deny "The technical writer writes only: ${DOC_PATHS[*]}."
        ;;
      *) deny "Unknown role '$role' in guard.sh." ;;
    esac
    ;;

  Read|Grep|Glob)
    # Only the test engineer has read restrictions: it must not see the implementation.
    [ "$role" = "test-engineer" ] || exit 0
    # A search rooted at the project, or at a folder that contains a source
    # folder, covers the source too. Treat it like a search with no path.
    if [ "$tool" != "Read" ]; then
      [ "$path" = "." ] || [ "$path" = "$PROJECT" ] && raw=""
      for d in "${SRC_DENY[@]}"; do [[ "$d" == "$path"/* ]] && raw=""; done
    fi
    if [ "$tool" != "Read" ] && [ -z "$raw" ]; then
      # A search with no path covers the whole repo. Allow a Glob only if its
      # pattern clearly starts outside the source folders.
      first="${pattern%%[*?\{]*}"
      if [ "$tool" = "Glob" ] && [ -n "$first" ] && ! in_any "${first%/}" "${SRC_DENY[@]}"; then
        exit 0
      fi
      deny "Pass an explicit path outside ${SRC_DENY[*]} when searching. You work from done.md, not the implementation."
    fi
    if in_any "$path" "${SRC_DENY[@]}"; then
      if [ "${#PUBLIC_ALLOW[@]}" -gt 0 ] && in_any "$path" "${PUBLIC_ALLOW[@]}"; then
        exit 0
      fi
      deny "The test engineer does not read the implementation (${SRC_DENY[*]}). Write tests from done.md and facts.md."
    fi
    ;;
esac

exit 0
