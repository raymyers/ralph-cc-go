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

- [x] Create `pkg/cminorsel/ast.go` with node interfaces
- [x] Define condition type (comparisons for branching)
- [x] Define addressing modes (target-specific):
  - [x] `Aindexed` (base + offset)
  - [x] `Aindexed2` (base + index)
  - [x] `Aindexed2shift` (base + index << shift) - ARM64 scaled addressing
  - [x] `Aindexed2ext` (base + extended index) - ARM64 sign/zero extended
  - [x] `Aglobal` (global symbol + offset)
  - [x] `Ainstack` (stack slot)
- [x] Define CminorSel expressions:
  - [x] `Evar`, `Econst` (like Cminor)
  - [x] `Eunop`, `Ebinop` (machine-level operators)
  - [x] `Eload` (with addressing mode)
  - [x] `Econdition` (conditional expression)
  - [x] `Elet`, `Eletvar` (let-binding with de Bruijn indices)
  - [x] `Eaddshift`, `Esubshift` (ARM64 combined operations)
- [x] Define CminorSel statements (similar to Cminor with extensions)
- [x] Add tests for AST construction

## Milestone 2: Machine Operators (Target-Independent)

**Goal:** Define the machine operator set

### Tasks

- [x] Create `pkg/cminorsel/ops.go`
- [x] Define unary operations:
  - [x] `Ocast8signed`, `Ocast8unsigned`, `Ocast16signed`, `Ocast16unsigned`
  - [x] `Onegint`, `Onotint`, `Onegf`, `Oabsf`
  - [x] `Osingleoffloat`, `Ofloatofsingle`
  - [x] Integer/float conversions
- [x] Define binary operations:
  - [x] `Oadd`, `Osub`, `Omul`, `Omulhs`, `Omulhu`
  - [x] `Odiv`, `Odivu`, `Omod`, `Omodu`
  - [x] `Oand`, `Oor`, `Oxor`, `Oshl`, `Oshr`, `Oshru`
  - [x] `Oaddf`, `Osubf`, `Omulf`, `Odivf`
  - [x] `Ocmp` comparisons
- [x] Define combined operations (if targeting ARM64):
  - [x] `Oaddshift`, `Osubshift` (shift + add/sub)
  - [x] `Omulaadd`, `Omulasub` (multiply-accumulate)
- [x] Add tests for operator definitions

**Notes:** Created `pkg/cminorsel/ops.go` with MachUnaryOp, MachBinaryOp, MachTernaryOp types. Includes ARM64-specific ops (rbit, clz, cls, rev, sqrt, combined shift+arith, fused multiply-accumulate, bic/orn/eon, conditional select). Helper methods: IsCommutative, IsCompare, IsShiftCombined, IsFusedMultiply.

## Milestone 3: Addressing Mode Selection

**Goal:** Recognize and select addressing modes

### Tasks

- [x] Create `pkg/selection/addressing.go`
- [x] Pattern matching for addressing modes:
  - [x] `base + constant` → `Aindexed`
  - [x] `base + index` → `Aindexed2`
  - [x] `base + index << shift` → `Aindexed2shift` (ARM64 scaled)
  - [x] `global + offset` → `Aglobal`
  - [x] `stackptr + offset` → `Ainstack`
- [x] Handle nested address computations
- [x] Target-specific mode availability (ARM64)
- [x] Add tests for addressing mode selection

**Notes:** Created `pkg/selection/addressing.go` with SelectAddressing function that pattern-matches Cminor address expressions to CminorSel addressing modes. Supports all ARM64 modes: Aglobal (symbol+offset), Ainstack (stack slot), Aindexed2shift (scaled array access with shifts 0-3), Aindexed2 (base+index), Aindexed (base+offset), with fallback to Aindexed{0}. Tests cover all patterns including commutative cases.

## Milestone 4: Operator Selection

**Goal:** Map Cminor operators to machine operators

### Tasks

- [x] Create `pkg/selection/ops.go`
- [x] Implement integer operation selection:
  - [x] Basic arithmetic (add, sub, mul, div)
  - [x] Shifts (shl, shr, shru)
  - [x] Bitwise (and, or, xor)
- [x] Implement floating-point operation selection
- [x] Implement comparison selection
- [x] Implement combined operation recognition:
  - [x] Shift + add → `Oaddshift`
  - [x] Load + op → memory operand (x86) — N/A for ARM64 target
- [x] Add tests for operator selection

**Notes:** Created `pkg/selection/ops.go` with SelectUnaryOp, SelectBinaryOp, TrySelectCombinedOp, and SelectComparison functions. Comprehensive tests cover all unary ops (negation, bitwise not, casts, conversions), all binary ops (arithmetic, bitwise, shifts, comparisons), combined shift+arith patterns (ARM64 addshift, subshift, andshift, orshift, xorshift for both int and long), and comparison helpers (NegateComparison, SwapComparison). Combined ops check both operand positions for commutative operations.

## Milestone 5: Expression Selection

**Goal:** Transform Cminor expressions to CminorSel

### Tasks

- [x] Create `pkg/selection/expr.go`
- [x] Select addressing modes for loads
- [x] Select operators for arithmetic
- [x] Handle sizeof/alignof (should be constants by now)
- [x] Implement if-conversion:
  - [x] Simple `if (c) x else y` → conditional move
  - [x] Check profitability of if-conversion
- [x] Add tests for expression selection

**Notes:** Created `pkg/selection/expr.go` with SelectionContext type holding globals and stack vars. Implements SelectExpr for all Cminor expression types (Evar, Econst, Eunop, Ebinop, Ecmp, Eload). Features: global vars become Oaddrsymbol, stack vars become Oaddrstack, combined shift+add/sub patterns recognized (ARM64 Eaddshift/Esubshift), loads use SelectAddressing for optimal addressing modes. Also implements SelectCondition for branch conditions and IsProfitableIfConversion heuristic for if-conversion decisions. Comprehensive tests cover all expression types, addressing modes, combined ops, conditions, and if-conversion profitability.

## Milestone 6: Statement Selection

**Goal:** Transform Cminor statements to CminorSel

### Tasks

- [x] Create `pkg/selection/stmt.go`
- [x] Transform stores with addressing mode selection
- [x] Transform conditionals with condition selection
- [x] Transform loops (mostly unchanged)
- [x] Handle function calls
- [x] Add tests for statement selection

**Notes:** Created `pkg/selection/stmt.go` with SelectStmt handling all Cminor statement types (Sskip, Sassign, Sstore, Scall, Stailcall, Sbuiltin, Sseq, Sifthenelse, Sloop, Sblock, Sexit, Sswitch, Sreturn, Slabel, Sgoto). Stores use SelectAddressing for optimal addressing modes. Conditionals use SelectCondition for proper condition type selection. Added SelectFunction and SelectProgram for full program transformation (populates globals map for addressing mode selection). Comprehensive tests cover all statement types including program-level global detection.

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
