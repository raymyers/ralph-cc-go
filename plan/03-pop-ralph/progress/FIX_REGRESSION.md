# Progress: Fix Regression Script Issues

## Task
Make `plan/06-regression-ralph/scripts/regression.sh` pass by investigating and fixing the issues.

## Original Status
4 failing seeds:
- crash:23463107 - still crashes
- crash:55976753 - still crashes  
- crash:928313049 - still crashes
- fail_compile:130805769 - now crashes (was only failing compile)

## Root Cause Analysis
All crashes were caused by the same bug: **assignment to function parameters**.

In C, function parameters can be modified (they're like local variables). The code:
```c
static int32_t func_2(uint8_t p_3, uint32_t p_4, int32_t p_5, int8_t p_6, uint8_t p_7) {
    for (p_7 = 0; (p_7 != 36); p_7 += 3) {
```

Was being compiled incorrectly:
1. `cshmgen/stmt.go:translateAssign` generated `Sstore(Eaddrof{Name: "p_7"}, value)` 
2. Since p_7 is a parameter (not in VarEnv), it became `Evar{Name: "p_7"}` as the address
3. RTL treated this as "store to address held in p_7's register"
4. At runtime: trying to write to address 0x1 (the parameter's value) → crash

## Fix
Modified `pkg/cshmgen` to handle parameter assignments correctly:

1. **Added parameter tracking to StmtTranslator** (`stmt.go`):
   - Track which names are parameters
   - For assignments to parameters, use `Sset` (temp assignment) instead of `Sstore`

2. **Added pre-scan for modified parameters** (`program.go`):
   - Before translation, scan function body to find which params are assigned
   - Allocate shadow temps for modified parameters
   - Generate initialization: copy original param values to temps at function start

3. **Updated ExprTranslator** (`expr.go`):
   - When reading a modified parameter, read from its shadow temp instead

This follows CompCert's approach where modified parameters are copied to local variables.

## Files Changed
- `pkg/cshmgen/stmt.go`: Added param tracking, use Sset for param assignments
- `pkg/cshmgen/expr.go`: Read modified params from shadow temps
- `pkg/cshmgen/program.go`: Pre-scan for modified params, generate init code

## Verification
- All 18 regression seeds now pass
- `make test` passes
- `make check` passes

## Status
COMPLETE
