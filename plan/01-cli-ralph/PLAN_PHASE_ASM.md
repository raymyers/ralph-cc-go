# Phase: Assembly Generation (Asmgen)

**Transformation:** Mach → Asm
**Prereqs:** Mach code generation (PLAN_PHASE_MACH.md)

Asm is the final target: actual assembly code for the target architecture.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `aarch64/Asm.v` | ARM64 assembly AST |
| `aarch64/Asmgen.v` | Mach to ARM64 translation |
| `aarch64/Asmgenproof.v` | Correctness proof |
| `aarch64/Asmgenproof0.v` | Auxiliary lemmas |
| `aarch64/PrintAsm.ml` | Assembly pretty-printer |
| `aarch64/TargetPrinter.ml` | Platform-specific output |

## Overview

Asmgen transforms Mach to assembly by:
1. **Instruction expansion** - Expand pseudo-ops to real instructions
2. **Address computation** - Compute addresses for globals
3. **Literal pools** - Handle large constants
4. **Assembly directives** - Add section markers, alignment

## Milestone 1: ARM64 Assembly AST Definition

**Goal:** Define ARM64 assembly instructions

**Status:** COMPLETE

### Tasks

- [x] Create `pkg/asm/ast.go` with instruction types
- [x] Define instruction classes:
  - [x] **Data processing:** ADD, SUB, AND, ORR, EOR, etc.
  - [x] **Shifts:** LSL, LSR, ASR, ROR
  - [x] **Multiply:** MUL, MADD, SMULL, UMULL
  - [x] **Divide:** SDIV, UDIV
  - [x] **Loads/Stores:** LDR, STR, LDP, STP
  - [x] **Branches:** B, BL, BR, BLR, RET
  - [x] **Conditionals:** B.cond, CSEL, CSET
  - [x] **Floating point:** FADD, FSUB, FMUL, FDIV, etc.
  - [x] **Conversions:** SCVTF, UCVTF, FCVTZS, FCVTZU
- [x] Define operand types:
  - [x] Register (W0-W30, X0-X30, D0-D31, S0-S31)
  - [x] Immediate
  - [x] Shifted register (not yet needed)
  - [x] Extended register (not yet needed)
  - [x] Memory (base, offset, indexing)
- [x] Define labels and symbols
- [x] Add tests for AST construction

## Milestone 2: Instruction Selection

**Goal:** Map Mach operations to ARM64 instructions

**Status:** COMPLETE

### Tasks

- [x] Create `pkg/asmgen/transform.go` (combined selection and transform)
- [x] Map integer operations:
  - [x] Oadd → ADD
  - [x] Osub → SUB
  - [x] Omul → MUL
  - [x] Odiv → SDIV/UDIV
  - [x] Oand → AND, Oor → ORR, Oxor → EOR
  - [x] Oshl → LSL, Oshr → ASR, Oshru → LSR
- [x] Map comparison operations:
  - [x] Ocmp → CMP + CSET
  - [x] Use condition codes: EQ, NE, LT, GE, etc.
- [x] Map floating-point operations:
  - [x] Direct mapping for most ops
- [ ] Map combined operations:
  - [ ] Oaddshift → ADD with shifted operand
  - [ ] Omadd → MADD
- [x] Add tests for instruction selection

## Milestone 3: Memory Operations

**Goal:** Generate load/store instructions

**Status:** COMPLETE (basic implementation)

### Tasks

- [x] Memory operations in `pkg/asmgen/transform.go`
- [x] Generate loads:
  - [x] LDRB, LDRH, LDR (W/X)
  - [x] Signed variants: LDRSB, LDRSH, LDRSW
  - [x] Floating: LDR (S/D)
- [x] Generate stores:
  - [x] STRB, STRH, STR (W/X)
  - [x] Floating: STR (S/D)
- [x] Handle addressing modes:
  - [x] Base + immediate offset
  - [ ] Base + register offset (partial)
  - [ ] Pre/post-indexed (for stack ops)
- [ ] Handle large offsets:
  - [ ] If offset doesn't fit, use temp register
- [x] Add tests for memory operations

## Milestone 4: Control Flow

**Goal:** Generate branch instructions

**Status:** COMPLETE (basic implementation)

### Tasks

- [x] Control flow in `pkg/asmgen/transform.go`
- [x] Generate unconditional branches:
  - [x] B label (short range)
  - [x] B.cond for conditional
- [x] Generate function calls:
  - [x] BL symbol (direct call)
  - [x] BLR Xn (indirect call)
- [x] Generate returns:
  - [x] RET (returns to LR)
- [x] Generate jump tables:
  - [x] Simplified compare-and-branch sequence
- [ ] Handle large branch offsets:
  - [ ] Use indirect branch if needed
- [x] Add tests for control flow

## Milestone 5: Constants and Literals

**Goal:** Handle constant materialization

**Status:** COMPLETE (basic implementation)

### Tasks

- [x] Constants in `pkg/asmgen/transform.go`
- [x] Small constants (0-65535):
  - [x] MOV with immediate
- [x] Medium constants:
  - [x] MOV + MOVK sequence
- [ ] Large constants:
  - [ ] Load from literal pool (using MOVZ+MOVK sequence for now)
- [x] Floating-point constants:
  - [x] FMOV immediate (simplified)
- [ ] Generate literal pool:
  - [ ] Place after function
  - [ ] Use PC-relative addressing
- [x] Add tests for constant handling

## Milestone 6: Assembly Output

**Goal:** Generate assembly text output

**Status:** COMPLETE

### Tasks

- [x] Create `pkg/asm/printer.go`
- [x] Generate section directives:
  - [x] `.text` for code
  - [x] `.data` for initialized data
  - [ ] `.bss` for uninitialized data
- [x] Generate function labels:
  - [x] `.global f`
  - [x] `.type f, @function`
  - [x] `f:`
- [x] Generate instructions:
  - [x] Proper operand formatting
  - [ ] Comments for readability
- [x] Generate data:
  - [x] `.byte`
  - [x] `.zero` for uninitialized
- [x] Handle alignment:
  - [x] `.align` directives
- [x] Add tests for assembly output

## Milestone 7: CLI Integration & Testing

**Goal:** Wire assembly generation to CLI, test against CompCert

**Status:** COMPLETE (basic integration)

### Tasks

- [x] Add `-dasm` flag implementation
- [x] Output to `.s` file
- [ ] Create test cases in `testdata/asm/`
- [ ] Create `testdata/asm.yaml` for parameterized tests
- [ ] Test against CompCert output (using container-use)
- [ ] Test assembly can be assembled (with `as`)
- [ ] Document any intentional deviations

## Milestone 8: End-to-End Testing

**Goal:** Test complete compilation pipeline

### Tasks

- [ ] Create `testdata/e2e/` with complete programs
- [ ] Test compilation: C → Asm
- [ ] Test assembly: Asm → object file (with `as`)
- [ ] Test linking: object → executable (with `ld` or `gcc`)
- [ ] Test execution: verify program behavior
- [ ] Compare against CompCert compiled code

## Test Strategy

1. **Unit tests:** Each instruction type
2. **Instruction encoding:** Verify valid ARM64
3. **Assembler tests:** Output assembles without errors
4. **Execution tests:** Compiled code produces correct results
5. **Golden tests:** Compare against CompCert's assembly

## Expected Output Format

ARM64 assembly output:
```asm
	.text
	.align	2
	.global	f
	.type	f, @function
f:
	sub	sp, sp, #32
	stp	x29, x30, [sp, #16]
	add	x29, sp, #16
	add	w0, w0, #1
	ldp	x29, x30, [sp, #16]
	add	sp, sp, #32
	ret
	.size	f, .-f
```

## ARM64 Architecture Notes

- 31 general registers: X0-X30 (64-bit), W0-W30 (32-bit)
- X31 is SP (stack pointer) or XZR (zero register)
- X29 is FP (frame pointer) by convention
- X30 is LR (link register)
- Immediate offsets in loads/stores are scaled
- Conditional execution uses flags from CMP/CMN/etc.

## Dependencies

- `pkg/mach` - Input AST (from PLAN_PHASE_MACH.md)
- ARM64 assembler for testing (as or gas)
- Target architecture (ARM64 aarch64-linux)
