# Phase: Cminor Generation (Cminorgen)

**Transformation:** Csharpminor → Cminor
**Prereqs:** Csharpminor generation (PLAN_PHASE_CSHARPMINOR.md)

Cminor introduces explicit stack allocation for address-taken local variables.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/Cminor.v` | Cminor AST definition |
| `cfrontend/Cminorgen.v` | Transformation from Csharpminor |
| `cfrontend/Cminorgenproof.v` | Correctness proof (study for semantics) |
| `backend/CminorSel.v` | Target of next phase (for context) |
| `backend/PrintCminor.ml` | OCaml pretty-printer for Cminor |

## Overview

Cminorgen transforms Csharpminor to Cminor by:
1. **Stack allocation** - Address-taken locals allocated on stack frame
2. **Switch simplification** - Complex switch → jump tables or if-cascades
3. **Memory model refinement** - Explicit stack pointer operations

## Milestone 1: Cminor AST Definition

**Goal:** Define the Cminor AST in Go

### Tasks

- [ ] Create `pkg/cminor/ast.go` with node interfaces
- [ ] Define Cminor expressions:
  - [ ] `Evar` (identifier - local or global)
  - [ ] `Econst` (integer, float, long, single)
  - [ ] `Eunop`, `Ebinop` (typed operators, similar to Csharpminor)
  - [ ] `Eload` (memory load with chunk)
- [ ] Define Cminor statements:
  - [ ] `Sskip`, `Sassign` (local variable assignment)
  - [ ] `Sstore` (memory store)
  - [ ] `Scall`, `Stailcall` (function calls)
  - [ ] `Sseq` (sequence)
  - [ ] `Sifthenelse`, `Sloop`, `Sblock`, `Sexit`
  - [ ] `Sswitch` (simplified switch)
  - [ ] `Sreturn`, `Slabel`, `Sgoto`
- [ ] Define function structure:
  - [ ] Function signature
  - [ ] Local variable declarations (stack-allocated)
  - [ ] Stack space requirement
  - [ ] Function body
- [ ] Define program structure
- [ ] Add tests for AST construction

## Milestone 2: Stack Frame Layout

**Goal:** Compute stack frame layout for functions

### Tasks

- [ ] Create `pkg/cminorgen/stack.go`
- [ ] Identify address-taken local variables
- [ ] Compute stack slot assignments
- [ ] Handle alignment requirements
- [ ] Calculate total stack size
- [ ] Generate stack access expressions
- [ ] Add tests for stack layout

## Milestone 3: Variable Transformation

**Goal:** Transform variable access for stack vs register

### Tasks

- [ ] Create `pkg/cminorgen/vars.go`
- [ ] Classify variables:
  - [ ] Register candidates (not address-taken)
  - [ ] Stack candidates (address-taken)
- [ ] Transform address-of operations to stack offsets
- [ ] Transform variable reads for stack variables to loads
- [ ] Transform variable writes for stack variables to stores
- [ ] Keep register variable access simple
- [ ] Add tests for variable transformation

## Milestone 4: Switch Statement Simplification

**Goal:** Transform switch statements to simpler forms

### Tasks

- [ ] Create `pkg/cminorgen/switch.go`
- [ ] Analyze switch case distribution
- [ ] Generate jump table for dense cases
- [ ] Generate binary search for sparse cases
- [ ] Generate linear if-cascade for small switches
- [ ] Handle default case properly
- [ ] Add tests for switch transformation

## Milestone 5: Statement and Expression Translation

**Goal:** Complete the transformation pass

### Tasks

- [ ] Create `pkg/cminorgen/transform.go`
- [ ] Translate expressions (mostly 1:1 from Csharpminor)
- [ ] Translate statements with stack variable handling
- [ ] Handle block scoping (Sblock → stack frame regions)
- [ ] Transform exit statements
- [ ] Handle function prologue/epilogue concepts
- [ ] Add tests for full transformation

## Milestone 6: CLI Integration & Testing

**Goal:** Wire Cminor generation to CLI, test against CompCert

### Tasks

- [ ] Add `-dcminor` flag implementation
- [ ] Create `pkg/cminor/printer.go` matching CompCert output format
- [ ] Create test cases in `testdata/cminor/`
- [ ] Create `testdata/cminor.yaml` for parameterized tests
- [ ] Test against CompCert output (using container-use)
- [ ] Document any intentional deviations

## Test Strategy

1. **Unit tests:** Stack layout, variable classification
2. **Integration tests:** Full pipeline from Csharpminor
3. **Golden tests:** Compare against CompCert's Cminor output
4. **Edge cases:** Complex switches, nested blocks, address-taken vars

## Expected Output Format

Cminor output should match CompCert's format:
```
"f"(x: int): int {
  stack 8
  var y;
  y = x + 1;
  return y;
}
```

## Notes

- Stack space is in bytes, computed at function level
- Address-taken distinction is crucial for correctness
- Switch simplification affects code quality significantly
- Cminor is the last "high-level" IR before instruction selection

## Dependencies

- `pkg/csharpminor` - Input AST (from PLAN_PHASE_CSHARPMINOR.md)
- `pkg/ctypes` - Type information
