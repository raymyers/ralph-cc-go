# Phase: CminorSel Generation (Selection)

**Transformation:** Cminor → CminorSel
**Prereqs:** Cminor generation (PLAN_PHASE_CMINOR.md)

CminorSel is Cminor with target-specific operators and addressing modes. This is the instruction selection phase.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/CminorSel.v` | CminorSel AST definition |
| `backend/Selection.v` | Instruction selection (generic) |
| `backend/SelectOp.v` | Operator selection interface |
| `backend/SelectLong.v` | 64-bit operation expansion |
| `aarch64/SelectOp.vp` | ARM64-specific operator selection |
| `x86/SelectOp.vp` | x86-specific operator selection |
| `backend/Op.v` | Machine-level operators |
| `backend/Machregs.v` | Machine register definitions |

## Overview

Selection transforms Cminor to CminorSel by:
1. **Instruction selection** - Map high-level ops to machine instructions
2. **Addressing mode recognition** - Identify complex addressing patterns
3. **Combined operations** - Fuse operations (e.g., shift+add)
4. **If-conversion** - Convert simple if/else to conditional moves
5. **64-bit expansion** - Split 64-bit ops on 32-bit targets

## Milestone 1: CminorSel AST Definition

**Goal:** Define the CminorSel AST with machine operations

### Tasks

- [ ] Create `pkg/cminorsel/ast.go` with node interfaces
- [ ] Define condition type (comparisons for branching)
- [ ] Define addressing modes (target-specific):
  - [ ] `Aindexed` (base + offset)
  - [ ] `Aindexed2` (base + index)
  - [ ] `Ascaled` (base + index * scale)
  - [ ] `Aglobal` (global symbol + offset)
  - [ ] `Ainstack` (stack slot)
- [ ] Define CminorSel expressions:
  - [ ] `Evar`, `Econst` (like Cminor)
  - [ ] `Eunop`, `Ebinop` (machine-level operators)
  - [ ] `Eload` (with addressing mode)
  - [ ] `Econdition` (conditional expression)
- [ ] Define CminorSel statements (similar to Cminor with extensions)
- [ ] Add tests for AST construction

## Milestone 2: Machine Operators (Target-Independent)

**Goal:** Define the machine operator set

### Tasks

- [ ] Create `pkg/cminorsel/ops.go`
- [ ] Define unary operations:
  - [ ] `Ocast8signed`, `Ocast8unsigned`, `Ocast16signed`, `Ocast16unsigned`
  - [ ] `Onegint`, `Onotint`, `Onegf`, `Oabsf`
  - [ ] `Osingleoffloat`, `Ofloatofsingle`
  - [ ] Integer/float conversions
- [ ] Define binary operations:
  - [ ] `Oadd`, `Osub`, `Omul`, `Omulhs`, `Omulhu`
  - [ ] `Odiv`, `Odivu`, `Omod`, `Omodu`
  - [ ] `Oand`, `Oor`, `Oxor`, `Oshl`, `Oshr`, `Oshru`
  - [ ] `Oaddf`, `Osubf`, `Omulf`, `Odivf`
  - [ ] `Ocmp` comparisons
- [ ] Define combined operations (if targeting ARM64):
  - [ ] `Oaddshift`, `Osubshift` (shift + add/sub)
  - [ ] `Omulaadd`, `Omulasub` (multiply-accumulate)
- [ ] Add tests for operator definitions

## Milestone 3: Addressing Mode Selection

**Goal:** Recognize and select addressing modes

### Tasks

- [ ] Create `pkg/selection/addressing.go`
- [ ] Pattern matching for addressing modes:
  - [ ] `base + constant` → `Aindexed`
  - [ ] `base + index` → `Aindexed2`
  - [ ] `base + index * scale` → `Ascaled` (x86)
  - [ ] `global + offset` → `Aglobal`
  - [ ] `stackptr + offset` → `Ainstack`
- [ ] Handle nested address computations
- [ ] Target-specific mode availability
- [ ] Add tests for addressing mode selection

## Milestone 4: Operator Selection

**Goal:** Map Cminor operators to machine operators

### Tasks

- [ ] Create `pkg/selection/ops.go`
- [ ] Implement integer operation selection:
  - [ ] Basic arithmetic (add, sub, mul, div)
  - [ ] Shifts (shl, shr, shru)
  - [ ] Bitwise (and, or, xor)
- [ ] Implement floating-point operation selection
- [ ] Implement comparison selection
- [ ] Implement combined operation recognition:
  - [ ] Shift + add → `Oaddshift`
  - [ ] Load + op → memory operand (x86)
- [ ] Add tests for operator selection

## Milestone 5: Expression Selection

**Goal:** Transform Cminor expressions to CminorSel

### Tasks

- [ ] Create `pkg/selection/expr.go`
- [ ] Select addressing modes for loads
- [ ] Select operators for arithmetic
- [ ] Handle sizeof/alignof (should be constants by now)
- [ ] Implement if-conversion:
  - [ ] Simple `if (c) x else y` → conditional move
  - [ ] Check profitability of if-conversion
- [ ] Add tests for expression selection

## Milestone 6: Statement Selection

**Goal:** Transform Cminor statements to CminorSel

### Tasks

- [ ] Create `pkg/selection/stmt.go`
- [ ] Transform stores with addressing mode selection
- [ ] Transform conditionals with condition selection
- [ ] Transform loops (mostly unchanged)
- [ ] Handle function calls
- [ ] Add tests for statement selection

## Milestone 7: CLI Integration & Testing

**Goal:** Wire instruction selection to CLI

### Tasks

- [ ] This is an internal phase (no CompCert dump flag)
- [ ] Create `pkg/cminorsel/printer.go` for debugging
- [ ] Create test cases in `testdata/cminorsel/`
- [ ] Add integration tests
- [ ] Test addressing mode selection thoroughly

## Test Strategy

1. **Unit tests:** Addressing mode and operator selection in isolation
2. **Pattern tests:** Test each pattern is recognized
3. **Integration tests:** Full pipeline from Cminor
4. **Target tests:** Test target-specific selections

## Notes

- Instruction selection is target-dependent
- Start with a simple subset (e.g., ARM64 or x86-64)
- Combined operations are important for performance
- If-conversion is optional but improves code quality
- This is the boundary between frontend and backend

## Target Considerations

For ARM64 (aarch64):
- Rich addressing modes
- Shift+op combinations
- Conditional instructions

For x86-64:
- Memory operands in most instructions
- Limited registers
- Complex addressing modes (base+index*scale+disp)

**Recommendation:** Start with ARM64 as it matches our CompCert build target.

## Dependencies

- `pkg/cminor` - Input AST (from PLAN_PHASE_CMINOR.md)
- Target architecture choice
