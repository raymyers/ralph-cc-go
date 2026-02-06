1. Check if there are any `csmith-reports/report-*.md` files not yet mentioned in `plan/05-fix-research-ralph/PLAN.md`. Use `plan/05-fix-research-ralph/scripts/new_csmith_findings.py` to do this.
2. Add any new finding filenames to `plan/05-fix-research-ralph/PLAN.md` as tasks to research, leave an unchecked tic-box.
3. Pick an item in `plan/05-fix-research-ralph/PLAN.md` to research.
4. Study `plan/05-fix-research-ralph/COMMON_CAUSES.md` other recent fixes (`plan/05-fix-research-ralph/fixes/` and then the source code, looking for the issue.
5. Write findings to a markdown named after the finding id:
  * If you find a fix plan it in `plan/05-fix-research-ralph/fixes/`
  * If you can't figure it out, write to `plan/05-fix-research-ralph/stuck/`
6. Update `plan/05-fix-research-ralph/COMMON_CAUSES.md`.

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
