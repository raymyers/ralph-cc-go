Execute these steps.

1. From the unfinished tasks in plan/02-cli-ralph/PLAN.md, choose a logical one to do next.
2. Do ONLY that task, and related automated tests.
3. Verify (including `make check`).
4. If complete: update PLAN to mark complete, commit.

## Environments

If you need to install things, use `container-use` MCP or docker.

## Tech Guidelines

We have a prototype C compiler, which we are trying to get working on real programs.

Our CLI is in Go lang, but following the compcert design with goal of equivalent output on each IR. Optimizations are not required (compare with -O0).

Makefile has test, lint and check (doing both).

For tests prefer data-driven from cases in in `testdata/*.yaml` listing input/output for examples. Also for e2e we can some full programs in `testdata/example-c.*.c`.

Docs have useful information, updated when needed.
