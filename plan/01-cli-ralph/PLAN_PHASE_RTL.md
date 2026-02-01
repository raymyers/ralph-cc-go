# Phase: RTL Generation (RTLgen)

**Transformation:** CminorSel → RTL
**Prereqs:** CminorSel generation (PLAN_PHASE_CMINORSEL.md)

RTL (Register Transfer Language) is the primary backend IR. It's a CFG-based representation with infinite pseudo-registers and 3-address code.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/RTL.v` | RTL AST definition |
| `backend/RTLgen.v` | Transformation from CminorSel |
| `backend/RTLgenproof.v` | Correctness proof |
| `backend/Registers.v` | Pseudo-register definitions |
| `backend/Op.v` | Machine operations (shared with CminorSel) |
| `backend/PrintRTL.ml` | OCaml pretty-printer for RTL |

## Overview

RTLgen transforms CminorSel to RTL by:
1. **Building CFG** - Convert structured code to control flow graph
2. **3-address code** - All operations have explicit destinations
3. **Pseudo-registers** - Infinite virtual registers
4. **Explicit control flow** - Branches and jumps between nodes

## Milestone 1: RTL AST Definition ✅

**Goal:** Define the RTL AST with CFG structure

### Tasks

- [x] Create `pkg/rtl/ast.go` with node interfaces
- [x] Define registers:
  - [x] Pseudo-register type (infinite supply)
  - [x] Register comparison/equality
- [x] Define RTL instructions:
  - [x] `Inop` (no operation, jump to successor)
  - [x] `Iop` (operation: `rd = op(rs...)`)
  - [x] `Iload` (memory load: `rd = Mem[addr]`)
  - [x] `Istore` (memory store: `Mem[addr] = rs`)
  - [x] `Icall` (function call)
  - [x] `Itailcall` (tail call)
  - [x] `Ibuiltin` (builtin operation)
  - [x] `Icond` (conditional branch)
  - [x] `Ijumptable` (indexed jump)
  - [x] `Ireturn` (function return)
- [x] Define CFG structure:
  - [x] Node type (positive integer)
  - [x] Instruction map (node → instruction)
  - [x] Entry point
- [x] Define function structure:
  - [x] Signature
  - [x] Parameters (as registers)
  - [x] CFG
  - [x] Entry node
- [x] Add tests for AST construction

**Notes:** Complete RTL AST with all instruction types, operations, conditions, and CFG structure. Tests verify interface implementations and instruction successors.

## Milestone 2: CFG Construction

**Goal:** Build control flow graph from structured code

### Tasks

- [ ] Create `pkg/rtlgen/cfg.go`
- [ ] Implement node allocation (fresh node IDs)
- [ ] Implement basic block construction:
  - [ ] Sequence of statements → chain of nodes
  - [ ] Last node links to exit
- [ ] Implement conditional translation:
  - [ ] `if (c) s1 else s2` → condition node + two branches
  - [ ] Merge point at end
- [ ] Implement loop translation:
  - [ ] Loop header node
  - [ ] Back edge to header
  - [ ] Exit edge
- [ ] Implement switch translation:
  - [ ] Jump table node
  - [ ] Case entry points
- [ ] Handle break/continue with proper exit edges
- [ ] Add tests for CFG construction

## Milestone 3: Register Allocation (Virtual)

**Goal:** Assign pseudo-registers to temporaries and expressions

### Tasks

- [ ] Create `pkg/rtlgen/regs.go`
- [ ] Implement fresh register generation
- [ ] Map CminorSel temporaries to registers
- [ ] Handle expression evaluation:
  - [ ] Subexpressions get temporaries
  - [ ] Results in designated registers
- [ ] Map function parameters to registers
- [ ] Handle return values
- [ ] Add tests for register assignment

## Milestone 4: Instruction Generation

**Goal:** Generate RTL instructions from CminorSel operations

### Tasks

- [ ] Create `pkg/rtlgen/instr.go`
- [ ] Generate `Iop` for expressions:
  - [ ] Binary ops: `rd = op(rs1, rs2)`
  - [ ] Unary ops: `rd = op(rs)`
  - [ ] Constants: `rd = const`
- [ ] Generate `Iload` for memory access:
  - [ ] Addressing mode → address operands
  - [ ] Destination register
- [ ] Generate `Istore` for memory writes:
  - [ ] Source register
  - [ ] Addressing mode → address operands
- [ ] Generate `Icall` for function calls:
  - [ ] Arguments in registers
  - [ ] Result register
  - [ ] Successor node
- [ ] Generate `Icond` for conditionals:
  - [ ] Condition operands
  - [ ] True/false successors
- [ ] Add tests for instruction generation

## Milestone 5: Expression Translation

**Goal:** Translate CminorSel expressions to RTL instruction sequences

### Tasks

- [ ] Create `pkg/rtlgen/expr.go`
- [ ] Implement expression evaluation order:
  - [ ] Left-to-right for arguments
  - [ ] Compute subexpressions first
- [ ] Handle nested expressions:
  - [ ] Generate temporaries
  - [ ] Chain instructions
- [ ] Handle addressof:
  - [ ] Stack slot address
  - [ ] Global address
- [ ] Handle conditional expressions:
  - [ ] Short-circuit evaluation
  - [ ] Join point
- [ ] Add tests for expression translation

## Milestone 6: Statement Translation

**Goal:** Translate CminorSel statements to RTL CFG

### Tasks

- [ ] Create `pkg/rtlgen/stmt.go`
- [ ] Translate assignment:
  - [ ] Evaluate RHS
  - [ ] Store to stack or keep in register
- [ ] Translate store:
  - [ ] Evaluate address
  - [ ] Evaluate value
  - [ ] Generate `Istore`
- [ ] Translate if/else:
  - [ ] Evaluate condition
  - [ ] Generate `Icond`
  - [ ] Recursively translate branches
- [ ] Translate loops:
  - [ ] Create header node
  - [ ] Generate body CFG
  - [ ] Back edge to header
- [ ] Translate function calls:
  - [ ] Evaluate arguments
  - [ ] Generate `Icall`
- [ ] Translate return:
  - [ ] Evaluate return expression
  - [ ] Generate `Ireturn`
- [ ] Add tests for statement translation

## Milestone 7: CLI Integration & Testing

**Goal:** Wire RTL generation to CLI, test against CompCert

### Tasks

- [ ] Add `-drtl` flag implementation
- [ ] Create `pkg/rtl/printer.go` matching CompCert output format
- [ ] Handle RTL dump numbering (`.rtl.0` for initial)
- [ ] Create test cases in `testdata/rtl/`
- [ ] Create `testdata/rtl.yaml` for parameterized tests
- [ ] Test against CompCert output (using container-use)
- [ ] Document any intentional deviations

## Test Strategy

1. **Unit tests:** CFG construction, instruction generation
2. **CFG validation:** Check all nodes reachable, proper successors
3. **Integration tests:** Full pipeline from CminorSel
4. **Golden tests:** Compare against CompCert's `.rtl.0`
5. **Edge cases:** Complex control flow, nested loops, switches

## Expected Output Format

RTL output should match CompCert's `.rtl.0` format:
```
f(x1) {
  4: x2 = Oadd(x1, 1) goto 3
  3: return x2
}
entry: 4
```

Format notes:
- Node numbers are positive integers
- Instructions show `nodenum: instruction goto successor` or `nodenum: instruction`
- Entry point shown at end
- Registers named `x1`, `x2`, etc.

## Notes

- RTL is the most important backend IR (optimizations happen here)
- CFG structure must be well-formed (all nodes reachable)
- Pseudo-registers are allocated fresh (no register pressure yet)
- This is where most optimizations will later be applied

## Dependencies

- `pkg/cminorsel` - Input AST (from PLAN_PHASE_CMINORSEL.md)
- `pkg/cminorsel/ops` - Machine operations
