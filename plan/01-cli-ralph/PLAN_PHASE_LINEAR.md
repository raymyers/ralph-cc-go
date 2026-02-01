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

## Milestone 1: Linear AST Definition ✓

**Goal:** Define the Linear AST with linear code

### Tasks

- [x] Create `pkg/linear/ast.go` with node interfaces
- [x] Define labels:
  - [x] Label type (positive integer)
  - [x] Label comparison/equality
- [x] Define Linear instructions:
  - [x] `Lgetstack` - Load from stack slot to register
  - [x] `Lsetstack` - Store register to stack slot
  - [x] `Lop` - Operation with locations
  - [x] `Lload` - Memory load
  - [x] `Lstore` - Memory store
  - [x] `Lcall` - Function call
  - [x] `Ltailcall` - Tail call
  - [x] `Lbuiltin` - Builtin operation
  - [x] `Llabel` - Label definition
  - [x] `Lgoto` - Unconditional jump
  - [x] `Lcond` - Conditional branch
  - [x] `Ljumptable` - Indexed jump
  - [x] `Lreturn` - Function return
- [x] Define function structure:
  - [x] Signature
  - [x] Stack size
  - [x] Code (instruction list)
- [x] Add tests for AST construction

## Milestone 2: CFG Linearization ✓

**Goal:** Flatten CFG to linear instruction sequence

### Tasks

- [x] Create `pkg/linearize/linearize.go`
- [x] Implement basic block ordering:
  - [x] Start from entry
  - [x] Follow fall-through when possible
  - [x] Reverse postorder traversal
- [x] Implement node enumeration:
  - [x] Depth-first postorder (reverse)
- [x] Convert LTL blocks to Linear instructions:
  - [x] Block start → label
  - [x] Block instructions → linear instructions
  - [x] Block terminator → branch/return
- [x] Handle fall-through optimization:
  - [x] If next block is target, omit jump
  - [x] Otherwise emit `Lgoto`
- [x] Handle conditional branches:
  - [x] Fall-through optimization for if-not branch
- [x] Add tests for linearization

## Milestone 3: Branch Tunneling ✓

**Goal:** Shortcut redundant jumps

### Tasks

- [x] Create `pkg/linearize/tunneling.go`
- [x] Build jump target map:
  - [x] For each label, what does it jump to?
  - [x] Handle chains: L1 → L2 → L3 becomes L1 → L3
- [x] Implement tunneling:
  - [x] `goto L1` where L1 is `goto L2` → `goto L2`
  - [x] Handle cycles (don't infinite loop)
- [x] Handle conditional tunneling:
  - [x] `if c then L1` where L1 is `goto L2` → `if c then L2`
- [x] Handle jump tables:
  - [x] Tunnel each table entry
- [x] Add tests for tunneling

## Milestone 4: Label Cleanup ✓

**Goal:** Remove unreferenced labels

### Tasks

- [x] Create `pkg/linearize/cleanup.go`
- [x] Collect used labels:
  - [x] All jump targets
  - [x] All conditional targets
  - [x] All jump table entries
- [x] Remove unused labels:
  - [x] Remove `Llabel` for unreferenced labels
- [x] Preserve entry point (always referenced)
- [x] Add tests for label cleanup

## Milestone 5: Stack Slot Assignment ✓

**Goal:** Prepare for activation record layout

### Tasks

- [x] Create `pkg/linearize/stack.go`
- [x] Collect all stack slot references (local, incoming, outgoing)
- [x] Compute stack frame size needed
- [x] Ensure 16-byte alignment (ARM64 requirement)
- [x] Add tests for stack assignment

## Milestone 6: CLI Integration & Testing ✓

**Goal:** Wire linearization to CLI (internal phase - no CLI flag)

### Tasks

- [x] This is an internal phase (no separate CompCert dump)
- [x] Create `pkg/linear/printer.go` for debugging
- [x] Unit tests for all components
- [x] Test tunneling effectiveness
- [x] Test label cleanup

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
