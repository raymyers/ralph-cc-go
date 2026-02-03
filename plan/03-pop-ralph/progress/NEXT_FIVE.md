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
| 2 | struct_point.c | Struct member access | TODO - need to create |
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

### Session 1 Continued: struct_point.c

Created `struct_point.c` to test struct member access.

**Bug Found:** Returns 64 instead of 42. Both struct fields are being stored/loaded at offset 0.

**Root Cause:** In Csharpminor, the struct is allocated with `var p[0]` (size 0) and both `p.x` and `p.y` compile to `int32[&p]` with no offset. The field offset calculation is missing.

**Location:** Bug is in the Clight to Csharpminor conversion (`pkg/cshmgen/`). The `generateFieldOffset` or similar logic is not computing offsets.

**Status:** DEFERRED - Struct support is a larger feature that needs dedicated attention.

### Summary

4 of 5 programs now work:
- recursive.c: ✅ FIXED
- global_var.c: ✅ Works
- many_args.c: ✅ Works  
- negative.c: ✅ Works
- struct_point.c: ❌ BUG - struct field offsets not computed

The main fix (callee-saved registers for params live across calls) resolved the recursive factorial issue. Struct support needs more work.

