# SQLite Setup Progress

## Status: COMPLETE

## What was done

1. **Verified existing checkouts** - Previous attempt already cloned:
   - `checkouts/sqlite/` - full SQLite repo
   - `checkouts/sqlite-amalgamation-3470200/` - amalgamation v3.47.2

2. **Added checkouts/ to .gitignore** - Keeps large external code out of version control

3. **Created plan/07-sqlite-ralph/PLAN.md** with 5 milestones:
   - Milestone 1: Preprocessing (handle system headers, __has_feature, __attribute__, etc.)
   - Milestone 2: Parsing (C99/C11 features: designated initializers, compound literals, etc.)
   - Milestone 3: Type Checking (int types, function pointers, incomplete types)
   - Milestone 4: Code Generation (linkage, large switches, varargs)
   - Milestone 5: Linking & Runtime (system libs, verify basic operations)

4. **Created supporting directories**:
   - `plan/07-sqlite-ralph/progress/` - for task progress files
   - `plan/07-sqlite-ralph/notes/` - for investigation notes

## Initial test

Running `./bin/ralph-cc -E checkouts/sqlite-amalgamation-3470200/sqlite3.c` shows:
```
#if: expected ')' in system header secure/_string.h
```

First blocker is preprocessor expression parsing in macOS SDK headers.

## Note on pre-existing test regression

`make check` shows one slow test failing (`C2.11 - array_access`). Bisect found this was introduced by commit 155bfdd before this task, not by these changes. Fast tests (`make test`) all pass.
