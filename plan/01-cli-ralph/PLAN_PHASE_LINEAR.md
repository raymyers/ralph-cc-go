# Phase: Linear Code Generation (Linearize)

**Transformation:** LTL → Linear
**Prereqs:** LTL generation (PLAN_PHASE_LTL.md)

Linear is linearized LTL with explicit labels and branches (no CFG). Also includes branch tunneling (Tunneling) and label cleanup (CleanupLabels).

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/Linear.v` | Linear AST definition |
| `backend/Linearize.v` | CFG linearization |
| `backend/Linearizeproof.v` | Correctness proof |
| `backend/Tunneling.v` | Branch tunneling optimization |
| `backend/CleanupLabels.v` | Remove unreferenced labels |
| `backend/PrintLinear.ml` | OCaml pretty-printer |

## Overview

This phase transforms the CFG-based LTL to a linear sequence:
1. **Linearize** - Flatten CFG to instruction sequence with labels
2. **Tunneling** - Shortcut jumps that go to other jumps
3. **CleanupLabels** - Remove unused labels

## Milestone 1: Linear AST Definition

**Goal:** Define the Linear AST with linear code

### Tasks

- [ ] Create `pkg/linear/ast.go` with node interfaces
- [ ] Define labels:
  - [ ] Label type (positive integer)
  - [ ] Label comparison/equality
- [ ] Define Linear instructions:
  - [ ] `Lgetstack` - Load from stack slot to register
  - [ ] `Lsetstack` - Store register to stack slot
  - [ ] `Lop` - Operation with locations
  - [ ] `Lload` - Memory load
  - [ ] `Lstore` - Memory store
  - [ ] `Lcall` - Function call
  - [ ] `Ltailcall` - Tail call
  - [ ] `Lbuiltin` - Builtin operation
  - [ ] `Llabel` - Label definition
  - [ ] `Lgoto` - Unconditional jump
  - [ ] `Lcond` - Conditional branch
  - [ ] `Ljumptable` - Indexed jump
  - [ ] `Lreturn` - Function return
- [ ] Define function structure:
  - [ ] Signature
  - [ ] Stack size
  - [ ] Code (instruction list)
- [ ] Add tests for AST construction

## Milestone 2: CFG Linearization

**Goal:** Flatten CFG to linear instruction sequence

### Tasks

- [ ] Create `pkg/linearize/linearize.go`
- [ ] Implement basic block ordering:
  - [ ] Start from entry
  - [ ] Follow fall-through when possible
  - [ ] Place frequently executed code first
- [ ] Implement node enumeration:
  - [ ] Depth-first postorder (reverse)
  - [ ] Or trace-based ordering
- [ ] Convert LTL blocks to Linear instructions:
  - [ ] Block start → label
  - [ ] Block instructions → linear instructions
  - [ ] Block terminator → branch/return
- [ ] Handle fall-through optimization:
  - [ ] If next block is target, omit jump
  - [ ] Otherwise emit `Lgoto`
- [ ] Handle conditional branches:
  - [ ] Try to make fall-through case "true"
  - [ ] May need to negate condition
- [ ] Add tests for linearization

## Milestone 3: Branch Tunneling

**Goal:** Shortcut redundant jumps

### Tasks

- [ ] Create `pkg/linearize/tunneling.go`
- [ ] Build jump target map:
  - [ ] For each label, what does it jump to?
  - [ ] Handle chains: L1 → L2 → L3 becomes L1 → L3
- [ ] Implement tunneling:
  - [ ] `goto L1` where L1 is `goto L2` → `goto L2`
  - [ ] Handle cycles (don't infinite loop)
- [ ] Handle conditional tunneling:
  - [ ] `if c then L1` where L1 is `goto L2` → `if c then L2`
- [ ] Handle jump tables:
  - [ ] Tunnel each table entry
- [ ] Iterate until no changes
- [ ] Add tests for tunneling

## Milestone 4: Label Cleanup

**Goal:** Remove unreferenced labels

### Tasks

- [ ] Create `pkg/linearize/cleanup.go`
- [ ] Collect used labels:
  - [ ] All jump targets
  - [ ] All conditional targets
  - [ ] All jump table entries
- [ ] Remove unused labels:
  - [ ] Remove `Llabel` for unreferenced labels
- [ ] Preserve entry point (always referenced)
- [ ] Add tests for label cleanup

## Milestone 5: Stack Slot Assignment

**Goal:** Prepare for activation record layout

### Tasks

- [ ] Create `pkg/linearize/stack.go` (or integrate with Stacking)
- [ ] Collect all stack slot references
- [ ] Compute stack frame size needed
- [ ] Ensure alignment requirements met
- [ ] Add tests for stack assignment

## Milestone 6: CLI Integration & Testing

**Goal:** Wire linearization to CLI

### Tasks

- [ ] This is an internal phase (no separate CompCert dump)
- [ ] Create `pkg/linear/printer.go` for debugging
- [ ] Create test cases in `testdata/linear/`
- [ ] Add integration tests
- [ ] Test tunneling effectiveness
- [ ] Test label cleanup

## Test Strategy

1. **Unit tests:** Linearization order, tunneling, cleanup
2. **CFG reconstruction:** Verify linear code has same semantics
3. **Fall-through:** Verify optimization reduces jumps
4. **Tunneling chains:** Test long jump chains
5. **Integration:** Full pipeline from LTL

## Notes

- Linearization order affects code locality
- Good ordering reduces jumps (fall-through)
- Tunneling is a simple but effective optimization
- Label cleanup is mostly cosmetic but reduces code size

## Branch Layout Strategies

### Simple (what we'll start with):
- Postorder traversal of CFG
- Fall-through for likely path

### Advanced (future):
- Profile-guided ordering
- Loop rotation
- Hot/cold splitting

## Dependencies

- `pkg/ltl` - Input AST (from PLAN_PHASE_LTL.md)
