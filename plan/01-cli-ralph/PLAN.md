[x] Add compcert submodule and get it to build https://github.com/AbsInt/CompCert
[x] Understand the order and meaning of the compcert phases / IRs, summarize in docs/PHASES.md
[x] Initialize a go with bin `ralph-cc`, cobra CLI, Makefile
[x] Implement placeholders (warn and exit) for debug flags: -dparse -dc -dasm -dclight -dcminor -drtl -dltl -dmach
[x] Study the Menhir grammar and determine the best plan for an equivelant parser in the Go CLI. You might choose a parsing lib, or recursive descent. Write your plan in docs/PARSING.md
[x] Study the plan in docs/PARSING.md, chose a tiny subset of C to try parsing, starting with tests driven by a `testdata/parse.yaml` input output, if approach needed to change update docs/PARSING.md.
[x] Make task bullets in plan/01-cli-ralph/PLAN_PARSING.md to carry out the plan in docs/PARSING.md, wiring to cli's -dparse, driven by the yaml tests, to ultimately reach equivelence on all supported syntax (-dparse matches).
[x] Get plan/01-cli-ralph/PLAN_PARSING.md to 25% tasks done
[x] Get plan/01-cli-ralph/PLAN_PARSING.md to 50% tasks done
[x] Get plan/01-cli-ralph/PLAN_PARSING.md to 75% tasks done
[x] Get plan/01-cli-ralph/PLAN_PARSING.md to 100% tasks done
[x] Ensure to cli's -dparse is wired to save that parsed AST in the same format as compcert saves. Create `testdata/example-c/all.c` to have a single exercise of all syntax.
[x] HEY! determine why this isn't dumping the parsed data `bin/ralph-cc testdata/example-c/all.c -dparse`
[x] Manually spot check our parser cli againse ccert (using container-use) and review tests for any gaps including checking for parser commit diffs that don't add test cases (adding task bullets to plan/01-cli-ralph/PLAN_TEST.md if needed).
[x] Execute plan/01-cli-ralph/PLAN_TEST.md, especially ensuring low duplication in test code and data is parameterized and yaml driven.
[x] Based on docs/PHASES.md, create a `plan/01-cli-ralph/PLAN_PHASE_*.MD` for each phase with an implementation plan, noting prereqs. Format as milestone sections with task bullets. Break it down for high assurance of success and solid data-driven testing. Ensure the key files from Compcert are called out to study (logic and AST structure...)
[ ] Implement the phase plan in `plan/01-cli-ralph/PLAN_PHASE_CLIGHT.md`, go incrementally (and and run tests as you go). Update phase plan marking your progress and notes if you get stuck. Bail if stuck, mark this complete if you finish.
