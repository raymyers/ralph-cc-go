# Next Five Test Programs

## Goal

Create 5 C programs that are likely to reveal compiler bugs, based on patterns from hello.c, fib.c, and fib_fn.c.

## Analysis of Previous Bugs

Previous programs revealed:
1. Stack frame layout issues (asmgen prologue/epilogue)
2. Outgoing slot offset issues for function calls
3. FP vs SP-relative addressing

## Predicted Problem Areas

1. **Recursion** - Stack growth, callee-saved registers, multiple frames
2. **Structs** - Field offsets, member access, struct as value
3. **Global variables** - Data section, symbol references
4. **Many arguments** - Stack argument passing (ARM64 >8 args)
5. **Signed arithmetic** - Sign extension, negative numbers

## Programs

| # | File | Feature | Status |
|---|------|---------|--------|
| 1 | recursive.c | Recursive factorial | ✅ FIXED - callee-saved register issue |
| 2 | struct_point.c | Struct member access | ✅ FIXED - struct field resolution |
| 3 | global_var.c | Global variable | ✅ Works |
| 4 | many_args.c | >8 function args | ✅ Works |
| 5 | negative.c | Negative number handling | ✅ Works |

## Progress

### Session 1: Recursive Fix

**Bug Found:** Parameters live across function calls were being allocated to caller-saved registers (X0-X7), causing values to be clobbered.

**Root Cause:** In `pkg/regalloc/irc.go`, the allocator was precoloring ALL parameters to their ABI-specified locations (X0, X1, etc.), even when those parameters were used after function calls.

**Fix:** Modified `NewAllocator` in `pkg/regalloc/irc.go` to check `graph.LiveAcrossCalls` before precoloring parameters. Parameters that are live across calls are NOT precolored, allowing them to be allocated to callee-saved registers (X19-X28).

**Tests Added:**
- `TestAnalyzeLivenessAcrossCall` in `pkg/regalloc/liveness_test.go`
- `TestRegisterLiveAcrossCallUsesCalleeSaved` in `pkg/regalloc/irc_test.go`

### Session 2: Struct Field Offset Fix

**Bug Found:** Returns 64 instead of 42. Both struct fields were being stored/loaded at offset 0.

**Root Cause:** In `pkg/clightgen/program.go`, when local variables were created in `collectLocalsFromStmt`, struct types were constructed via `TypeFromString("struct Point")` which only sets the name but not the fields. The struct definition (with fields) was tracked separately but not resolved when creating local variable types.

The result was:
- Clight `p.x` and `p.y` showed correct field access
- But the local variable `p` had type `Tstruct{Name: "Point", Fields: []}` (0 fields)
- Csharpminor showed `var p[0]` (size 0) and `int32[&p]` for both fields

**Fix:** Modified `collectLocalsFromStmt` in `pkg/clightgen/program.go` to resolve struct types using `simplExpr.ResolveStruct()` after getting the type from `TypeFromString`. This populates the `Fields` slice from the registered struct definitions.

Changes made:
- `pkg/clightgen/program.go`: Added struct resolution in DeclStmt and For loop cases

After fix:
- Csharpminor shows `var p[8]` (correct size)
- Field x: `int32[&p]` (offset 0)
- Field y: `int32[addl(&p, 4L)]` (offset 4)
- Assembly shows `str w0, [x29]` and `str w0, [x29, #4]`
- Returns 42 (10 + 32) as expected

### Summary

All 5 programs now work:
- recursive.c: ✅ Returns 120 (factorial(5))
- struct_point.c: ✅ Returns 42 (10 + 32)
- global_var.c: ✅ Returns 50 (42 + 8)
- many_args.c: ✅ Returns 45 (1+2+3+4+5+6+7+8+9)
- negative.c: ✅ Returns 42 (abs(-42))

