# Agent team workflow: spec (draft)

Nov 10, 2026 · @Mathew Waddilove

## Purpose and principles

A team of Claude Code agents takes a task from agreed spec to a branch that is ready for your review, while you direct traffic and decide only what needs you. It has two goals: a high standard of output, and decisions surfaced to you promptly instead of guessed at silently.

- **Code routes, agents work.** Who runs next is decided by a fixed routing table and enforced by hooks, not by agent judgement.
- **Independence over role count.** A check adds value only if it runs in a fresh context, with a different brief, ideally grounded in something executable. Add a role only when a real failure shows the need.
- **Done is defined before work starts.** As much of it as possible is executable, and no agent can relax it.
- **Ambiguity is escalated, not resolved.** Any agent can block the task and put a decision in your queue.
- **State lives on disk.** Each agent starts fresh and reads the task folder; nothing depends on conversation history. The one exception is a role that raised a blocker: the delivery lead resumes that same agent with its context intact once you answer, and a fresh agent can still take over from the task folder if the resume fails.
- **Ceremony scales with risk.** Small, well-covered changes skip most of the pipeline (see Tiers).

## Roles

Six agent roles plus you, named after the matching jobs on a real dev team. Every role can also raise a blocker and return `blocked`.

| Role | Responsible for | Reads | Writes | Must not touch | Tools |
| --- | --- | --- | --- | --- | --- |
| You (engineering manager) | Approving `done.md`, answering blockers, merging the branch | Everything | `resolutions/`, final approval |  |  |
| Product owner | Spec, `done.md`, facts, initial tier; works with you | Repo, your input | `done.md`, `facts.md`, `state.json` (tier) | Code, tests | Read, Grep, Glob, Write (task folder only) |
| Delivery lead | Dispatching the next role, retries, tier step-ups, writing the summary commit | Task folder | `state.json`, the final commit message | `done.md`, tests, code, blockers raised by other roles, `resolutions/` | Agent(listed roles), Read, Bash (git, gh) |
| Test engineer | Acceptance tests from `done.md`, written to fail first | `done.md`, `facts.md`, public interfaces | `tests/acceptance/` | Implementation (no reading it either) | Read, Grep, Glob, Write, Edit, Bash |
| Developer | Implementation and unit tests until all checks pass | Task folder, repo | Source, unit tests | `done.md`, `tests/acceptance/` | Read, Grep, Glob, Write, Edit, Bash |
| Reviewer | Test coverage of every outcome line, probing for gaps, ruling on judged criteria, and on tier 2, checking the docs against the code | Task folder, diff, repo | `findings.md`, `verdict.json` | Everything else | Read, Grep, Glob, Bash (read-only commands) |
| Technical writer (tier 2) | Docs and ADRs from the diff and the task folder, before review | Diff, `done.md`, `findings.md`, `blockers/`, `resolutions/` | Docs, ADRs | Code, tests | Read, Grep, Glob, Write, Edit |

The delivery lead may retry a step, re-route, or answer a question that `done.md` or `facts.md` already answers. It may not interpret an ambiguous spec, change scope, waive a finding, or step a tier down.

## Workflow

&#91;embedded content: tier 2 workflow · 6 stages, 3 fault loops, 1 escalation path\]

The delivery lead dispatches each stage in order and routes on `verdict.json`: a code fault goes back to the developer, a test fault to the test engineer, and a doc fault to the technical writer. Tier 1 skips the test engineer and technical writer; tier 0 is the developer alone.

1. The product owner drafts `done.md` with you. Once you approve it with `scripts/approve.sh`, it locks, and the task moves into its own worktree, named after the task folder, on branch `task/<name>`. Every later stage runs there. Pass `--start` to start the delivery lead in the new worktree. The delivery lead commits after each stage, with the task number as a prefix, for example `[002] Record review round 1: pass`.
2. The test engineer writes acceptance tests that fail against the current code.
3. The developer implements until every executable check passes.
4. The technical writer updates the docs.
5. The reviewer checks coverage, the judged criteria and the docs, then writes a verdict.
6. On a fail, the owner of each blocking finding fixes it, the technical writer brings the docs in line with the fixes, and the reviewer runs again, up to three rounds. On a pass, the delivery lead makes a final commit whose message describes the change, for use as the pull request description, and a hook notifies you that the task is ready. The process record (checks, rounds, findings, blockers, tier) stays in the task folder.

## Tiers

The product owner proposes a tier with you when `done.md` is agreed. The delivery lead can step a task up on its own; stepping down needs your approval.

| Tier | Use when | Pipeline | `done.md` |
| --- | --- | --- | --- |
| 0 | Existing checks decide correctness, one reasonable implementation, diff reviewable by you in minutes, cheap to undo | Developer only, with the test suite as a Stop hook | Not needed; the task line is the spec |
| 1 | Some judgement, low blast radius | Developer, then reviewer | Short: outcome, checks, escalation triggers |
| 2 | Ambiguous, or touches auth, payments, migrations, public APIs or data deletion, or any outcome needs an acceptance test | Full pipeline, including test engineer and technical writer | Full template |

Promote a task to the next tier when it produces an escalation, a reviewer finding above minor, or a diff noticeably larger than expected. Size in lines is not a criterion: a one-line change to auth is tier 2.

Acceptance tests belong to tier 2 alone. On tier 1, a unit or service-level test covers an outcome line. If a tier 1 line can only be checked by an acceptance test, the reviewer raises a blocking finding against the tier, and the delivery lead steps the task up to tier 2.

## Artifacts

Each task gets a folder that holds all shared state. Agents read it at start and write only their own files.

```
.tasks/042-login-rate-limit/
  done.md         # product owner; locked once you approve it
  facts.md        # product owner; rules, constraints, known answers
  state.json      # delivery lead; stage, tier, round counts
  findings.md     # reviewer; append-only
  verdict.json    # reviewer; latest verdict
  blockers/       # one file per question; any role adds one, nobody edits one
  resolutions/    # one file per answer, same name as its blocker; only you write here
tests/acceptance/ # test engineer
```

**`done.md`** has these sections: Outcome, Executable checks (commands that must all exit 0), Judged criteria, Out of scope, Escalate if. The worked example from our earlier discussion (rate-limiting `POST /login`) is the reference.

**A blocker** is one file, `blockers/D-003.md`, in a fixed shape so you can answer quickly:

```markdown
# D-003 · raised by developer · 2026-11-10

Question: The acceptance test expects 429 after 5 attempts; done.md says "more than 5". Which?
Options: (a) block on the 6th attempt (b) block on the 5th
Recommendation: (a), matching the outcome text; the test needs changing.
Evidence: done.md line 4; tests/acceptance/login-rate-limit.test.ts:31
```

You answer with `scripts/answer.sh D-003 "<answer>"`, which prints the blocker and writes your answer to `resolutions/D-003.md`. With no id, it answers the only open blocker, and with no answer it opens `$EDITOR`. A blocker is open while no resolution file has its name, so no file ever changes after it is written. The guard hook refuses an agent write to `resolutions/` and to a blocker that already exists.

**`verdict.json`** is what the delivery lead routes on:

```json
{
  "status": "pass | fail | blocked",
  "round": 2,
  "findings": [
    {"id": "F-4", "severity": "blocking | minor", "owner": "developer | test-engineer",
     "criterion": "Judged criterion 2", "evidence": "src/login.ts:88"}
  ]
}
```

Only `blocking` findings stop the task. Minor findings stay in `findings.md` for you to see.

## Enforcement in Claude Code

Each role is a file in `.claude/agents/`, with its tool list from the Roles table. The delivery lead runs as the main session (`claude --agent delivery-lead`), and its `tools: Agent(product-owner, test-engineer, developer, reviewer, technical-writer)` line limits what it can dispatch. Anything that must not be skipped is a hook, not an instruction.

| Rule | Hook | Scope | Action on failure |
| --- | --- | --- | --- |
| `done.md` and `facts.md` are locked after approval | PreToolUse on Edit, Write | All agents | Exit 2: block, tell the agent to raise a blocker |
| Acceptance tests are read-only after the test engineer finishes | PreToolUse on Edit, Write for `tests/acceptance/` | Developer, reviewer | Exit 2 |
| Test engineer cannot read the implementation | PreToolUse on Read, Grep for source paths | Test engineer | Exit 2 |
| Developer cannot finish with failing checks | SubagentStop, matched to developer | Developer | Run the executable checks from `done.md`; exit 2 with the failures |
| Reviewer must leave a valid verdict | SubagentStop, matched to reviewer | Reviewer | Validate `verdict.json` against the schema; exit 2 if invalid |
| Delivery lead cannot end without an outcome | Stop | Delivery lead | Block unless a pass verdict plus a final commit at stage `ready`, or an open blocker, exists |
| You hear about blocks | Notification and Stop | Delivery lead | Desktop or phone notification, `<NNN> <role>: <status>` with no body, for example `002 delivery-lead: blocked D-001` |

Every Stop and SubagentStop hook checks `stop_hook_active` and lets the agent stop on a second attempt, so a broken check cannot loop forever. Round caps live in `state.json`: after three failed rounds on one stage, the delivery lead escalates instead of retrying.

## Escalation triggers

An agent must raise a blocker and return `blocked` when any of these happen. Everything else it handles itself.

- The spec is ambiguous, or two lines of `done.md` conflict.
- A test seems to contradict `done.md`.
- The work needs something listed as out of scope, or a file outside the expected area.
- A new dependency, schema change, or public API change is needed and `done.md` doesn't cover it.
- An action is irreversible (data deletion, a migration, anything pushed outside the repo).
- A stage fails three rounds in a row.
- Anyone wants to lower the tier or relax `done.md`.

The delivery lead surfaces each blocker unedited. It can give its own recommendation in its message to you, but only you answer a blocker. When you answer, the delivery lead resumes the agent that raised it, so the question costs no re-reading. `state.json` records which agent that is.

## Open questions

- [ ] Is the product owner an agent in a chat with you, or a session you start by hand for each task?
- [x] Should the developer work in a git worktree, so parallel tasks don't collide? Yes: the whole pipeline for a task runs in one worktree, named after the task folder.
- [x] Does the delivery lead open the PR itself, or leave a branch for you to open it? Neither, for now: it leaves a branch whose final commit message describes the change, and you are notified that it is ready.
- [ ] Is three rounds the right cap, and should it differ by stage?
- [ ] Where do notifications go: desktop, phone, or a Slack channel?
- [ ] At what point does the delivery lead become a shell script calling each role headlessly (`claude -p --agent …`)? `--agent` works with `-p`: a probe session ran that way.
- [ ] Should one task's implementation be split across parallel developers, one per service? They could run as parallel subagents, or as an agent team so they can coordinate while they work. Before choosing a team: confirm that hooks in project settings, keyed on `agent_type`, and a `TeammateIdle` hook enforce the developer's rules for teammates, since teammates skip their definition's `hooks`. Either way, each slice's writable paths are enforced, each slice gets its own checks, and `checks.sh` stays the gate for the whole task. A change to the contract between services is a blocker. Acceptance tests stay before code, and unit tests stay with the developer.
