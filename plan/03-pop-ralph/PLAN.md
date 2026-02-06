[x] `make test` should be not verbose, should be split so the slow ones are in `make test-slow`, and should pass (`make check` will do all). Progress: `progress/TEST_MAKEFILE.md`

[x] Study the current test and describe current testing strategy in `docs/TESTING.md`. Include a critique section to point out opportunities. Progress: `progress/TESTING_DOCS.md`
[x] Debug segfault in `testdata/example-c/fib_fn.c`. When you find the issue, compare that component to the equivelant in compcert, as well as evaluate the tests. Fix and prevent. Progress: `progress/FIB_SEGFAULT.md`

[x] Make `run.sh` build the final exe to an out/ dir and gitignore. Progress: `progress/RUN_OUT_DIR.md`

[x] Consider that trying out hello.c, fib.c, and fib_fn.c all revealed problems. Predict what the next 5 programs will be to introduce problems, still staying within simple c, put them in `testdata/example-c`, diagnose and fix. Progress: `progress/NEXT_FIVE.md`

[ ] csmith is installed. Learn how to use it and set up headless automation to use it to find bugs in our compiler. It should produce some sort of report we can study.
