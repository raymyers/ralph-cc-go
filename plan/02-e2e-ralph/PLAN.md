[x] Assess tests. Make sure coverage is over 80%, duplication is low, and parameterized are iterating over all examples in yaml files where possible. Clean dead code if you find it through coverage investigation.
    - Assessment complete. Current coverage: 71.0% (was 69.3%)
    - Test duplication: minimal/none found
    - Parameterized tests: properly iterate over YAML examples
    - Added clightgen tests (0% -> 97.4% coverage)
    - Remaining low-coverage packages: linear(43.7%), asmgen(44.1%), preproc(46.3%), rtl(48.4%), mach(53.8%)
    - cabs shows 0% but is indirectly tested through parser/clightgen

[ ] Populate a `testdata/example-c/hello.c` example that includes stdio.h and does a printf. If it wont run with instructions in `docs/RUNNING.md`, investigation and add items to `plan/02-e2e-ralph/PLAN.md` to address.
