[x] `make test` should be not verbose, should be split so the slow ones are in `make test-slow`, and should pass (`make check` will do all). Progress: `progress/TEST_MAKEFILE.md`

[x] Study the current test and describe current testing strategy in `docs/TESTING.md`. Include a critique section to point out opportunities. Progress: `progress/TESTING_DOCS.md`
[x] Debug segfault in `testdata/example-c/fib_fn.c`. When you find the issue, compare that component to the equivelant in compcert, as well as evaluate the tests. Fix and prevent. Progress: `progress/FIB_SEGFAULT.md`

[x] Make `run.sh` build the final exe to an out/ dir and gitignore. Progress: `progress/RUN_OUT_DIR.md`

[x] Consider that trying out hello.c, fib.c, and fib_fn.c all revealed problems. Predict what the next 5 programs will be to introduce problems, still staying within simple c, put them in `testdata/example-c`, diagnose and fix. Progress: `progress/NEXT_FIVE.md`

[x] Based on `plan/04-learn/ANALYSIS.md`, update AGENTS.md and supporting docs, and make other changes that seem appropriate. Progress: `progress/AGENTS_DOCS_UPDATE.md`

[x] csmith is installed. Learn how to use it and set up headless automation to use it to find bugs in our compiler. It should produce some sort of report we can study. Progress: `progress/CSMITH_FUZZER.md`

[x] Apply all fixes in `plan/05-fix-research-ralph/fixes` and confirm they fix the issue (if possible by re-running csmith with the same seed.) Progress: `progress/APPLY_CSMITH_FIXES.md`

[ ] Where possible, address common causes systemically that are in `plan/05-fix-research-ralph/COMMON_CAUSES.md`, at design, type or testing level, make the codebase safer while continuing to pass all tests.
