# Acceptance tests

This folder holds a stand-in for a real acceptance suite, so the agent team pipeline has something to run. Only the test engineer writes here.

## Conventions

- One file per task, named `<task-folder>.test.sh`, for example `001-list-skus.test.sh`.
- A test file is a Bash script that exits 0 when every check in it passes, and non-zero otherwise.
- Write one check per Outcome line in the task's `done.md`, and print the line it covers before you run it.
- Test through public interfaces only: an HTTP endpoint, a command's output, or a file a command produces.
- `run.sh` runs every `*.test.sh` file and fails if any of them fails.
