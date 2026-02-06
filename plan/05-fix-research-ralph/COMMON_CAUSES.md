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

## Categories to Watch

1. **Stack layout** - offset calculations, frame pointer usage
2. **Type coercions** - signed/unsigned, width conversions
3. **Operator semantics** - division, shifts, overflow
4. **Control flow** - branch conditions, fall-through
5. **ABI compliance** - calling conventions, register usage
6. **AST completeness** - handle all node types (Paren, Cast, etc.)
7. **Global initialization** - Init bytes propagation through IR passes
