# Common Causes of Compilation Bugs

## Stack Frame Layout Issues

### Callee-Save Register Offset Sign Error (CONFIRMED)

**Symptom**: Runtime crashes (SIGSEGV/SIGBUS) in functions using callee-saved registers (x19-x28).

**Cause**: Callee-saved registers stored at positive offsets from FP, which go outside the allocated stack frame into invalid memory.

**Location**: `pkg/stacking/layout.go` CalleeSaveOffset calculation

**Fix**: Use negative offsets from FP for callee-saved register storage.

---

## AST Node Type Coverage

### Missing Paren Handling (CONFIRMED: mismatch_263236830)

**Symptom**: Parenthesized expressions silently fail or produce wrong results. E.g., `static int8_t g_2 = (-3);` initializes to 0 instead of -3.

**Cause**: `evaluateConstantInitializer` handles `Constant` and `Unary` but not `Paren`.

**Location**: `pkg/clightgen/program.go` evaluateConstantInitializer

**Fix**: Add `case cabs.Paren:` that recursively processes `e.Expr`

### Missing Typedef Resolution

**Symptom**: Typedef names like `int8_t` become default `int` type.

**Location**: `pkg/clightgen/types.go` TypeFromString

**Fix**: Add cases for standard integer typedefs (`int8_t`, `uint16_t`, etc.)

---

## Register Allocation & Spilling

### Incomplete Stack Slot Handling in Stacking Transform (CONFIRMED: fail_compile_130805769)

**Symptom**: Panic "stack slot in register position - regalloc incomplete" during assembly generation.

**Cause**: Register allocator correctly spills high-pressure variables to stack slots. However, `pkg/stacking/transform.go` only handles stack slots for `Lop` instructions. When spilled variables appear in `Lload.Dest`, `Lstore.Src`, `Lcond.Args`, etc., `locToReg()` panics.

**Location**: `pkg/stacking/transform.go` - `transformInst` for Lload, Lstore, Lcond, Ljumptable, Lbuiltin

**Fix**: Add spill/reload handling for all instruction types that access `Loc` fields, similar to how `transformLop` already handles it for operations.

**Trigger**: Functions with >10 variables live across function calls (exhausts callee-saved registers X19-X28).

---

## Operator Semantics

### Logical AND/OR Missing Short-Circuit (CONFIRMED: mismatch_2487828851)

**Symptom**: Boolean expressions with `&&` or `||` produce wrong values. E.g., `(-8 && -8)` returns `-8` instead of `1`.

**Cause**: `cabsToBinaryOp()` has placeholder code that maps logical operators (`OpAnd`/`OpOr`) to bitwise operators (`Oand`/`Oor`).

**Location**: `pkg/simplexpr/transform.go` - `transformBinary()` and `cabsToBinaryOp()`

**Fix**: Add `case cabs.OpAnd:` and `case cabs.OpOr:` in `transformBinary()` that convert to short-circuit conditional evaluation:
- `a && b` → `if (a) { if (b) 1 else 0 } else 0`
- `a || b` → `if (a) 1 else { if (b) 1 else 0 }`

---

## Categories to Watch

1. **Stack layout** - offset calculations, frame pointer usage
2. **Type coercions** - signed/unsigned, width conversions
3. **Operator semantics** - division, shifts, overflow
4. **Control flow** - branch conditions, fall-through
5. **ABI compliance** - calling conventions, register usage
6. **AST completeness** - handle all node types (Paren, Cast, etc.)
7. **Global initialization** - Init bytes propagation through IR passes
8. **Incomplete implementations** - partial fixes that reference missing code
