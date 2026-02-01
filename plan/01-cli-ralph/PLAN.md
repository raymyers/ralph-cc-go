[x] Add compcert submodule and get it to build https://github.com/AbsInt/CompCert
[x] Understand the order and meaning of the compcert phases / IRs, summarize in docs/PHASES.md
[x] Initialize a go with bin `ralph-cc`, cobra CLI, Makefile
[ ] Implement placeholders (warn and exit) for debug flags: -dparse -dc -dasm -dclight -dcminor -drtl -dltl -dmach
[ ] Study the Menhir grammar and determine the best plan for an equivelant parser in the Go CLI. You might choose a parsing lib, or recursive descent. Write your plan in docs/PARSING.md
[ ] Study the plan in docs/PARSING.md, chose a tiny subset of C to try parsing, starting with tests driven by a `testdata/parse.yaml` input output, if approach needed to change update docs/PARSING.md.
[ ] Updated docs/PARSING.md with a bulleted list of syntax elements needed for complete equivelance.
