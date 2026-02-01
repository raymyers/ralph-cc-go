# Phase: Mach Code Generation (Stacking)

**Transformation:** Linear → Mach
**Prereqs:** Linear code generation (PLAN_PHASE_LINEAR.md)

Mach is a near-assembly representation with concrete activation record layout. This is the last intermediate language before assembly.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/Mach.v` | Mach AST definition |
| `backend/Stacking.v` | Activation record layout |
| `backend/Stackingproof.v` | Correctness proof |
| `backend/Bounds.v` | Stack frame size computation |
| `aarch64/Stacklayout.v` | ARM64 stack layout |
| `backend/PrintMach.ml` | OCaml pretty-printer |
| `aarch64/Machregs.v` | Machine register definitions |

## Overview

Stacking transforms Linear to Mach by:
1. **Frame layout** - Compute activation record structure
2. **Stack access** - Replace abstract slots with concrete offsets
3. **Prologue/epilogue** - Add function entry/exit code
4. **Callee-save handling** - Save/restore preserved registers

## Milestone 1: Mach AST Definition

**Goal:** Define the Mach AST with concrete stack layout

### Tasks

- [ ] Create `pkg/mach/ast.go` with node interfaces
- [ ] Define machine registers (same as LTL)
- [ ] Define Mach instructions:
  - [ ] `Mgetstack` - Load from stack at concrete offset
  - [ ] `Msetstack` - Store to stack at concrete offset
  - [ ] `Mgetparam` - Load parameter from caller's frame
  - [ ] `Mop` - Operation
  - [ ] `Mload` - Memory load
  - [ ] `Mstore` - Memory store
  - [ ] `Mcall` - Function call
  - [ ] `Mtailcall` - Tail call
  - [ ] `Mbuiltin` - Builtin
  - [ ] `Mlabel` - Label
  - [ ] `Mgoto` - Unconditional jump
  - [ ] `Mcond` - Conditional branch
  - [ ] `Mjumptable` - Indexed jump
  - [ ] `Mreturn` - Return
- [ ] Define function structure:
  - [ ] Code
  - [ ] Stack frame size
  - [ ] Used callee-save registers
- [ ] Add tests for AST construction

## Milestone 2: Stack Frame Layout

**Goal:** Compute concrete stack frame layout

### Tasks

- [ ] Create `pkg/stacking/layout.go`
- [ ] Define frame structure (ARM64):
  ```
  +---------------------------+  <- old SP (caller's frame)
  | Return address (LR)       |
  | Saved FP                  |
  +---------------------------+  <- FP
  | Callee-saved registers    |
  | Local variables           |
  | Outgoing arguments        |
  +---------------------------+  <- SP (16-byte aligned)
  ```
- [ ] Compute frame sections:
  - [ ] Callee-save area size
  - [ ] Local variable area size
  - [ ] Outgoing argument area size
- [ ] Handle alignment:
  - [ ] 16-byte stack alignment (ARM64)
  - [ ] Per-variable alignment
- [ ] Compute total frame size
- [ ] Add tests for layout computation

## Milestone 3: Stack Slot Translation

**Goal:** Translate abstract slots to concrete offsets

### Tasks

- [ ] Create `pkg/stacking/slots.go`
- [ ] Map Local slots to frame offsets
- [ ] Map Outgoing slots to bottom of frame
- [ ] Map Incoming slots to caller's frame:
  - [ ] Above our frame pointer
  - [ ] Depends on calling convention
- [ ] Generate stack access instructions:
  - [ ] `Lgetstack Local` → `Mgetstack fp+offset`
  - [ ] `Lsetstack Local` → `Msetstack fp+offset`
  - [ ] `Lgetstack Incoming` → `Mgetparam offset`
- [ ] Add tests for slot translation

## Milestone 4: Callee-Save Register Handling

**Goal:** Save and restore callee-saved registers

### Tasks

- [ ] Create `pkg/stacking/calleesave.go`
- [ ] Identify used callee-saved registers:
  - [ ] ARM64: X19-X28, D8-D15
  - [ ] Scan function for uses
- [ ] Compute save/restore locations:
  - [ ] Sequential in callee-save area
  - [ ] Paired stores for ARM64 (STP/LDP)
- [ ] Generate prologue saves:
  - [ ] At function entry
  - [ ] After frame setup
- [ ] Generate epilogue restores:
  - [ ] Before return
  - [ ] Before tail call
- [ ] Add tests for callee-save handling

## Milestone 5: Prologue and Epilogue

**Goal:** Generate function entry and exit code

### Tasks

- [ ] Create `pkg/stacking/prolog.go`
- [ ] Generate prologue:
  - [ ] Save link register (return address)
  - [ ] Save frame pointer
  - [ ] Set up new frame pointer
  - [ ] Allocate stack frame
  - [ ] Save callee-saved registers
- [ ] Generate epilogue:
  - [ ] Restore callee-saved registers
  - [ ] Restore frame pointer
  - [ ] Deallocate stack frame
  - [ ] Return (restore PC from LR)
- [ ] Handle leaf functions:
  - [ ] May omit frame pointer setup
  - [ ] May skip saving LR if not used
- [ ] Add tests for prologue/epilogue

## Milestone 6: Instruction Translation

**Goal:** Translate Linear instructions to Mach

### Tasks

- [ ] Create `pkg/stacking/transform.go`
- [ ] Translate stack operations with concrete offsets
- [ ] Translate other instructions (mostly unchanged)
- [ ] Insert prologue at function entry
- [ ] Insert epilogue before returns
- [ ] Handle tail calls (epilogue before call)
- [ ] Add tests for instruction translation

## Milestone 7: CLI Integration & Testing

**Goal:** Wire Mach generation to CLI, test against CompCert

### Tasks

- [ ] Add `-dmach` flag implementation
- [ ] Create `pkg/mach/printer.go` matching CompCert output format
- [ ] Create test cases in `testdata/mach/`
- [ ] Create `testdata/mach.yaml` for parameterized tests
- [ ] Test against CompCert output (using container-use)
- [ ] Document any intentional deviations

## Test Strategy

1. **Unit tests:** Frame layout, slot translation
2. **Stack correctness:** Verify offsets are correct
3. **Callee-save:** Verify all used regs saved/restored
4. **Alignment:** Verify 16-byte alignment maintained
5. **Golden tests:** Compare against CompCert's `-dmach`

## Expected Output Format

Mach output should match CompCert's `.mach` format:
```
f:
  sub sp, sp, #32
  stp fp, lr, [sp, #16]
  add fp, sp, #16
  ...
  ldp fp, lr, [sp, #16]
  add sp, sp, #32
  ret
```

## ARM64 Frame Notes

- FP (X29) points into frame
- SP must be 16-byte aligned
- LR (X30) contains return address
- Pairs of registers stored with STP/LDP

## Dependencies

- `pkg/linear` - Input AST (from PLAN_PHASE_LINEAR.md)
- Target architecture (ARM64)
