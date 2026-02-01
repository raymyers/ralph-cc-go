[x] Add compcert submodule and get it to build https://github.com/AbsInt/CompCert
[x] Understand the order and meaning of the compcert phases / IRs, summarize in docs/PHASES.md
[x] Initialize a go with bin `ralph-cc`, cobra CLI, Makefile
[x] Implement placeholders (warn and exit) for debug flags: -dparse -dc -dasm -dclight -dcminor -drtl -dltl -dmach
[x] Study the Menhir grammar and determine the best plan for an equivelant parser in the Go CLI. You might choose a parsing lib, or recursive descent. Write your plan in docs/PARSING.md
[x] Study the plan in docs/PARSING.md, chose a tiny subset of C to try parsing, starting with tests driven by a `testdata/parse.yaml` input output, if approach needed to change update docs/PARSING.md.
[x] Make task bullets in plan/01-cli-ralph/PLAN_PARSING.md to carry out the plan in docs/PARSING.md, wiring to cli's -dparse, driven by the yaml tests, to ultimately reach equivelence on all supported syntax (-dparse matches).
[x] Get plan/01-cli-ralph/PLAN_PARSING.md to 25% tasks done
[ ] Get plan/01-cli-ralph/PLAN_PARSING.md to 50% tasks done
[ ] Get plan/01-cli-ralph/PLAN_PARSING.md to 75% tasks done
[ ] Get plan/01-cli-ralph/PLAN_PARSING.md to 100% tasks done
