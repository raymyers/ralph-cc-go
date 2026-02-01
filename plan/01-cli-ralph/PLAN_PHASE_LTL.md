# Phase: LTL Generation (Allocation)

**Transformation:** RTL → LTL
**Prereqs:** RTL generation (PLAN_PHASE_RTL.md)

LTL (Location Transfer Language) replaces pseudo-registers with physical registers and stack slots. This is the register allocation phase.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/LTL.v` | LTL AST definition |
| `backend/Allocation.v` | Register allocator interface |
| `backend/Allocproof.v` | Correctness proof |
| `backend/Locations.v` | Location definitions (regs + stack) |
| `backend/Conventions.v` | Calling conventions |
| `aarch64/Conventions1.v` | ARM64 calling convention |
| `backend/PrintLTL.ml` | OCaml pretty-printer for LTL |
| `backend/Regalloc.ml` | Register allocator (OCaml, uses IRC) |

## Overview

Register allocation transforms RTL to LTL by:
1. **Liveness analysis** - Determine when registers are live
2. **Interference graph** - Build graph of conflicting registers
3. **Graph coloring** - Assign physical registers
4. **Spilling** - Handle overflow with stack slots
5. **Calling convention** - Place arguments/returns correctly

## Milestone 1: LTL AST Definition

**Goal:** Define the LTL AST with physical locations

### Tasks

- [ ] Create `pkg/ltl/ast.go` with node interfaces
- [ ] Define locations:
  - [ ] `R(r)` - Physical register `r`
  - [ ] `S(Local, ofs, ty)` - Local stack slot
  - [ ] `S(Incoming, ofs, ty)` - Incoming stack argument
  - [ ] `S(Outgoing, ofs, ty)` - Outgoing stack argument
- [ ] Define machine registers for target (ARM64):
  - [ ] `X0`-`X30` (integer registers)
  - [ ] `D0`-`D31` (floating-point registers)
  - [ ] Special: `SP`, `LR`, etc.
- [ ] Define LTL instructions:
  - [ ] `Lnop`, `Lop`, `Lload`, `Lstore`
  - [ ] `Lcall`, `Ltailcall`
  - [ ] `Lbuiltin`
  - [ ] `Lbranch`, `Lcond`, `Ljumptable`
  - [ ] `Lreturn`
  - [ ] All use locations instead of pseudo-registers
- [ ] Define basic blocks (not single instructions)
- [ ] Add tests for AST construction

## Milestone 2: Liveness Analysis

**Goal:** Compute liveness information for RTL

### Tasks

- [ ] Create `pkg/regalloc/liveness.go`
- [ ] Implement dataflow equations:
  - [ ] `live_out[n] = ∪ live_in[s]` for successors s
  - [ ] `live_in[n] = use[n] ∪ (live_out[n] - def[n])`
- [ ] Compute use/def sets for each instruction
- [ ] Implement fixed-point iteration
- [ ] Handle function parameters (live at entry)
- [ ] Handle return values (live at return)
- [ ] Add tests for liveness analysis

## Milestone 3: Interference Graph

**Goal:** Build interference graph from liveness

### Tasks

- [ ] Create `pkg/regalloc/interference.go`
- [ ] Build interference edges:
  - [ ] Two registers interfere if both live at same point
  - [ ] Special: defined register interferes with all live-out
- [ ] Handle preference edges (for coalescing):
  - [ ] Move instructions create preferences
  - [ ] Call arguments/returns create preferences
- [ ] Build affinity edges for move coalescing
- [ ] Add tests for interference graph

## Milestone 4: Graph Coloring (Iterated Register Coalescing)

**Goal:** Implement register allocator using IRC algorithm

### Tasks

- [ ] Create `pkg/regalloc/irc.go`
- [ ] Implement simplify phase:
  - [ ] Remove low-degree non-move nodes
  - [ ] Push to stack
- [ ] Implement coalesce phase:
  - [ ] Merge preference-related nodes (George/Briggs)
  - [ ] Conservative coalescing
- [ ] Implement freeze phase:
  - [ ] Give up on coalescing for some moves
- [ ] Implement potential spill selection:
  - [ ] Select high-degree nodes
  - [ ] Spill cost heuristics
- [ ] Implement select phase:
  - [ ] Pop stack, assign colors
  - [ ] Handle actual spills
- [ ] Implement spill code insertion:
  - [ ] Generate load before use
  - [ ] Generate store after def
- [ ] Add tests for IRC algorithm

## Milestone 5: Calling Convention

**Goal:** Handle argument and return value placement

### Tasks

- [ ] Create `pkg/regalloc/conventions.go`
- [ ] Define ARM64 calling convention:
  - [ ] Integer args: `X0`-`X7`, then stack
  - [ ] Float args: `D0`-`D7`, then stack
  - [ ] Return: `X0` (int), `D0` (float)
- [ ] Handle caller-saved registers:
  - [ ] `X0`-`X18` are caller-saved
  - [ ] Must be saved around calls if live
- [ ] Handle callee-saved registers:
  - [ ] `X19`-`X28` are callee-saved
  - [ ] Must be saved in prologue if used
- [ ] Compute stack frame for arguments
- [ ] Add tests for calling convention

## Milestone 6: RTL to LTL Translation

**Goal:** Complete the transformation pass

### Tasks

- [ ] Create `pkg/regalloc/transform.go`
- [ ] Apply register assignment to instructions
- [ ] Replace pseudo-registers with locations
- [ ] Insert spill code (loads/stores)
- [ ] Handle moves between locations:
  - [ ] Reg-to-reg: simple move
  - [ ] Reg-to-stack: store
  - [ ] Stack-to-reg: load
  - [ ] Stack-to-stack: via temp register
- [ ] Group instructions into basic blocks
- [ ] Add tests for transformation

## Milestone 7: CLI Integration & Testing

**Goal:** Wire register allocation to CLI, test against CompCert

### Tasks

- [ ] Add `-dltl` flag implementation
- [ ] Create `pkg/ltl/printer.go` matching CompCert output format
- [ ] Create test cases in `testdata/ltl/`
- [ ] Create `testdata/ltl.yaml` for parameterized tests
- [ ] Test against CompCert output (using container-use)
- [ ] Document any intentional deviations

## Test Strategy

1. **Unit tests:** Liveness, interference graph, IRC phases
2. **Correctness:** Verify no conflicting registers assigned same location
3. **Golden tests:** Compare against CompCert's `-dltl`
4. **Stress tests:** Functions with high register pressure
5. **Calling convention:** Test argument/return placement

## Expected Output Format

LTL output should match CompCert's `.ltl` format:
```
f(X0) {
  4: { Lop(Oadd, [X0; #1], X1); Lbranch 3 }
  3: { Lreturn (Some X1) }
}
entry: 4
```

Format notes:
- Locations shown as `X0`, `D0` (registers) or stack slots
- Instructions grouped in blocks
- Basic blocks separated

## Notes

- Register allocation is the most complex backend pass
- IRC is the standard algorithm (used by many compilers)
- Spilling quality affects performance significantly
- Calling convention correctness is critical

## Validation

CompCert uses a verified validator (not verified allocator):
- Allocator implemented in OCaml (untrusted)
- Coq validator checks result (trusted)
- We can do the same: implement simple allocator, validate result

## Dependencies

- `pkg/rtl` - Input AST (from PLAN_PHASE_RTL.md)
- Target architecture (ARM64 for now)
