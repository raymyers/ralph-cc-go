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
| 1 | recursive.c | Recursive factorial | TODO |
| 2 | struct_point.c | Struct member access | TODO |
| 3 | global_var.c | Global variable | TODO |
| 4 | many_args.c | >8 function args | TODO |
| 5 | negative.c | Negative number handling | TODO |

## Progress

### Current: Starting

