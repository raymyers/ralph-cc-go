# Phase: Clight Generation (SimplExpr + SimplLocals)

**Transformation:** CompCert C (Cabs) → Clight
**Prereqs:** Completed parser (cabs package)

This is the first major transformation pass. It takes the parsed C AST (Cabs) and produces Clight, which is C without side-effects in expressions.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `cfrontend/Clight.v` | Clight AST definition |
| `cfrontend/SimplExpr.v` | Main transformation: pull side-effects out of expressions |
| `cfrontend/SimplLocals.v` | Pull non-addressable scalar locals out of memory |
| `cfrontend/Cop.v` | C operators and their semantics |
| `cfrontend/Ctypes.v` | C type system |
| `lib/Integers.v` | Integer representations |
| `exportclight/Clightgen.ml` | OCaml pretty-printer for Clight |

## Overview

Clight simplifies CompCert C by:
1. **Fixing evaluation order** - C's undefined evaluation order becomes deterministic
2. **Eliminating expression side-effects** - All side-effects (assignments, increments) become statements
3. **Simplifying locals** - Non-addressable scalars are pulled out of memory

## Milestone 1: Clight AST Definition

**Goal:** Define the Clight AST in Go following the CompCert structure

### Tasks

- [ ] Create `pkg/clight/ast.go` with Clight node interfaces
- [ ] Define Clight expressions (subset of C - no side-effects):
  - [ ] `Econst_int`, `Econst_float`, `Econst_long`, `Econst_single`
  - [ ] `Evar` (variable reference)
  - [ ] `Etempvar` (temporary variable - key difference from Cabs)
  - [ ] `Ederef` (pointer dereference)
  - [ ] `Eaddrof` (address-of)
  - [ ] `Eunop`, `Ebinop` (unary/binary operators)
  - [ ] `Ecast` (type cast)
  - [ ] `Efield` (struct field access)
  - [ ] `Esizeof`, `Ealignof`
- [ ] Define Clight statements:
  - [ ] `Sskip`, `Sassign`, `Sset` (temp assignment)
  - [ ] `Scall` (function call as statement)
  - [ ] `Sbuiltin` (builtin call)
  - [ ] `Ssequence` (statement sequence)
  - [ ] `Sifthenelse`, `Sloop`, `Sbreak`, `Scontinue`
  - [ ] `Sreturn`, `Sswitch`, `Slabel`, `Sgoto`
- [ ] Define function and program structures
- [ ] Add tests for AST construction

## Milestone 2: Type System Foundation

**Goal:** Implement minimal type representation needed for Clight

### Tasks

- [ ] Create `pkg/ctypes/types.go` with C type definitions
- [ ] Define basic types: `Tvoid`, `Tint`, `Tfloat`, `Tlong`
- [ ] Define composite types: `Tpointer`, `Tarray`, `Tfunction`
- [ ] Define struct/union types: `Tstruct`, `Tunion`
- [ ] Add type attributes (signedness, size)
- [ ] Implement type comparison/equality
- [ ] Add tests for type operations

## Milestone 3: SimplExpr Transformation

**Goal:** Transform Cabs expressions to Clight, extracting side-effects

### Tasks

- [ ] Create `pkg/simplexpr/transform.go`
- [ ] Implement expression classification:
  - [ ] Identify pure expressions (no side-effects)
  - [ ] Identify expressions with side-effects
- [ ] Implement temporary variable generation
- [ ] Transform assignment expressions (`=`, `+=`, etc.) to statements
- [ ] Transform increment/decrement (`++`, `--`) to statements
- [ ] Transform function calls in expressions to temporaries
- [ ] Transform comma operator to statement sequence
- [ ] Transform ternary operator with side-effects
- [ ] Handle compound assignments correctly
- [ ] Add comprehensive tests from `testdata/simplexpr.yaml`

## Milestone 4: SimplLocals Transformation

**Goal:** Optimize local variable handling

### Tasks

- [ ] Create `pkg/simpllocals/transform.go`
- [ ] Identify address-taken locals (need to stay in memory)
- [ ] Identify non-addressable scalar locals (can use temps)
- [ ] Transform stack locals to temporaries where possible
- [ ] Update variable references accordingly
- [ ] Add tests for local optimization

## Milestone 5: CLI Integration & Testing

**Goal:** Wire Clight generation to CLI, test against CompCert

### Tasks

- [ ] Add `-dclight` flag implementation in CLI
- [ ] Create `pkg/clight/printer.go` matching CompCert output format
- [ ] Create `testdata/clight/` directory with test cases
- [ ] Create `testdata/clight.yaml` for parameterized tests
- [ ] Test against CompCert output (using container-use)
- [ ] Document any intentional deviations from CompCert

## Test Strategy

1. **Unit tests:** Test each transformation in isolation
2. **Integration tests:** Full pipeline from C source to Clight
3. **Golden tests:** Compare output against CompCert's `-dclight`
4. **Edge cases:** Focus on complex expression side-effects

## Expected Output Format

Clight output should match CompCert's `.light.c` format:
```c
/* Clight generated from source.c */
void f(int x) {
  int t1;
  t1 = g(x);  /* function call becomes statement */
  return t1 + 1;
}
```

## Dependencies

- `pkg/cabs` - Input AST (✅ complete)
- `pkg/lexer` - Lexical analysis (✅ complete)
- `pkg/parser` - Parsing (✅ complete)

## Notes

- Temporary variables use names like `$1`, `$2` etc.
- Evaluation order is strictly left-to-right in Clight
- SimplLocals is optional but important for efficiency
