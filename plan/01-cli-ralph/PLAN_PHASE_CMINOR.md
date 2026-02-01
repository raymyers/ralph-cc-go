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

- [x] Create `pkg/cminor/ast.go` with node interfaces
- [x] Define Cminor expressions:
  - [x] `Evar` (identifier - local or global)
  - [x] `Econst` (integer, float, long, single)
  - [x] `Eunop`, `Ebinop` (typed operators, similar to Csharpminor)
  - [x] `Eload` (memory load with chunk)
- [x] Define Cminor statements:
  - [x] `Sskip`, `Sassign` (local variable assignment)
  - [x] `Sstore` (memory store)
  - [x] `Scall`, `Stailcall` (function calls)
  - [x] `Sseq` (sequence)
  - [x] `Sifthenelse`, `Sloop`, `Sblock`, `Sexit`
  - [x] `Sswitch` (simplified switch)
  - [x] `Sreturn`, `Slabel`, `Sgoto`
- [x] Define function structure:
  - [x] Function signature
  - [x] Local variable declarations (stack-allocated)
  - [x] Stack space requirement
  - [x] Function body
- [x] Define program structure
- [x] Add tests for AST construction

**Notes:** Re-exported shared types (Chunk, UnaryOp, BinaryOp, Comparison) from csharpminor to avoid duplication. Key differences from Csharpminor: Sassign uses variable names instead of temp IDs, Scall result is a variable name, Functions have Stackspace and Vars fields.

## Milestone 2: Stack Frame Layout

**Goal:** Compute stack frame layout for functions

### Tasks

- [x] Create `pkg/cminorgen/stack.go`
- [x] Identify address-taken local variables
- [x] Compute stack slot assignments
- [x] Handle alignment requirements
- [x] Calculate total stack size
- [x] Generate stack access expressions
- [x] Add tests for stack layout

**Notes:** Implemented StackLayout, StackSlot types with ComputeStackLayout function. Natural alignment rules: 1-byte for char, 2-byte for short, 4-byte for int, 8-byte for long/pointer. Total frame aligned to 8 bytes (aarch64). FindAddressTaken scans statement tree for Eaddrof expressions referencing locals.

## Milestone 3: Variable Transformation

**Goal:** Transform variable access for stack vs register

### Tasks

- [x] Create `pkg/cminorgen/vars.go`
- [x] Classify variables:
  - [x] Register candidates (not address-taken)
  - [x] Stack candidates (address-taken)
- [x] Transform address-of operations to stack offsets
- [x] Transform variable reads for stack variables to loads
- [x] Transform variable writes for stack variables to stores
- [x] Keep register variable access simple
- [x] Add tests for variable transformation

**Notes:** Implemented VarEnv with ClassifyVariables that uses FindAddressTaken to determine which vars need stack allocation. VarKind enum for VarRegister/VarStack classification. TransformAddrOf generates constant offset expressions, TransformVarRead generates Evar for registers or Eload for stack, TransformVarWrite generates Sassign for registers or Sstore for stack.

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
