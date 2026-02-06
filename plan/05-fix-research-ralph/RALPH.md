Execute these steps.

1. Check if there are any `csmith-reports/report-*.md` files not yet mentioned in `plan/05-fix-research-ralph/PLAN.md`. Use `plan/05-fix-research-ralph/scripts/new_csmith_findings.py` to do this.
2. Add any new finding filenames to `plan/05-fix-research-ralph/PLAN.md` as tasks to research, leave an unchecked tic-box.
3. Pick a random item in `plan/05-fix-research-ralph/PLAN.md` to research.
4. Study `plan/05-fix-research-ralph/COMMON_CAUSES.md` other recent fixes (`plan/05-fix-research-ralph/fixes/`
5. Investigate the issue in the source code.
5. Write findings to a markdown named after the finding id:
  * If you find a fix plan it in `plan/05-fix-research-ralph/fixes/`
  * If you can't figure it out, write to `plan/05-fix-research-ralph/stuck/`
6. Update `plan/05-fix-research-ralph/COMMON_CAUSES.md` if relevant.
7. Commit your progress, local to `plan/05-fix-research-ralph`.

## Make No Changes

You may run code to investigate but do not change it. Only record ideas for us to make changes later.

## Finding Index

Looks like `20260205-123456`.

## UV scripts

Use this pattern to make self contained scripts.
```
#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
```

## Tech Guidelines

We have a prototype C compiler, which we are trying to get working on real programs.

Our CLI is in Go lang, but following the compcert design with goal of equivalent output on each IR. Optimizations are not required (compare with -O0).

Makefile has test, lint and check (doing both).
