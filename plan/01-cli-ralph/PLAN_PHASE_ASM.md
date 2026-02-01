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

### Tasks

- [ ] Create `pkg/asm/ast.go` with instruction types
- [ ] Define instruction classes:
  - [ ] **Data processing:** ADD, SUB, AND, ORR, EOR, etc.
  - [ ] **Shifts:** LSL, LSR, ASR, ROR
  - [ ] **Multiply:** MUL, MADD, SMULL, UMULL
  - [ ] **Divide:** SDIV, UDIV
  - [ ] **Loads/Stores:** LDR, STR, LDP, STP
  - [ ] **Branches:** B, BL, BR, BLR, RET
  - [ ] **Conditionals:** B.cond, CSEL, CSET
  - [ ] **Floating point:** FADD, FSUB, FMUL, FDIV, etc.
  - [ ] **Conversions:** SCVTF, UCVTF, FCVTZS, FCVTZU
- [ ] Define operand types:
  - [ ] Register (W0-W30, X0-X30, D0-D31, S0-S31)
  - [ ] Immediate
  - [ ] Shifted register
  - [ ] Extended register
  - [ ] Memory (base, offset, indexing)
- [ ] Define labels and symbols
- [ ] Add tests for AST construction

## Milestone 2: Instruction Selection

**Goal:** Map Mach operations to ARM64 instructions

### Tasks

- [ ] Create `pkg/asmgen/select.go`
- [ ] Map integer operations:
  - [ ] Oadd → ADD
  - [ ] Osub → SUB
  - [ ] Omul → MUL
  - [ ] Odiv → SDIV/UDIV
  - [ ] Oand → AND, Oor → ORR, Oxor → EOR
  - [ ] Oshl → LSL, Oshr → ASR, Oshru → LSR
- [ ] Map comparison operations:
  - [ ] Ocmp → CMP + CSET
  - [ ] Use condition codes: EQ, NE, LT, GE, etc.
- [ ] Map floating-point operations:
  - [ ] Direct mapping for most ops
- [ ] Map combined operations:
  - [ ] Oaddshift → ADD with shifted operand
  - [ ] Omadd → MADD
- [ ] Add tests for instruction selection

## Milestone 3: Memory Operations

**Goal:** Generate load/store instructions

### Tasks

- [ ] Create `pkg/asmgen/memory.go`
- [ ] Generate loads:
  - [ ] LDRB, LDRH, LDR (W/X)
  - [ ] Signed variants: LDRSB, LDRSH, LDRSW
  - [ ] Floating: LDR (S/D)
- [ ] Generate stores:
  - [ ] STRB, STRH, STR (W/X)
  - [ ] Floating: STR (S/D)
- [ ] Handle addressing modes:
  - [ ] Base + immediate offset
  - [ ] Base + register offset
  - [ ] Pre/post-indexed (for stack ops)
- [ ] Handle large offsets:
  - [ ] If offset doesn't fit, use temp register
- [ ] Add tests for memory operations

## Milestone 4: Control Flow

**Goal:** Generate branch instructions

### Tasks

- [ ] Create `pkg/asmgen/branch.go`
- [ ] Generate unconditional branches:
  - [ ] B label (short range)
  - [ ] B.cond for conditional
- [ ] Generate function calls:
  - [ ] BL symbol (direct call)
  - [ ] BLR Xn (indirect call)
- [ ] Generate returns:
  - [ ] RET (returns to LR)
- [ ] Generate jump tables:
  - [ ] ADR to get table address
  - [ ] LDR to load offset
  - [ ] BR to jump
- [ ] Handle large branch offsets:
  - [ ] Use indirect branch if needed
- [ ] Add tests for control flow

## Milestone 5: Constants and Literals

**Goal:** Handle constant materialization

### Tasks

- [ ] Create `pkg/asmgen/constants.go`
- [ ] Small constants (0-65535):
  - [ ] MOV with immediate
- [ ] Medium constants:
  - [ ] MOV + MOVK sequence
- [ ] Large constants:
  - [ ] Load from literal pool
- [ ] Floating-point constants:
  - [ ] FMOV if representable
  - [ ] Load from literal pool otherwise
- [ ] Generate literal pool:
  - [ ] Place after function
  - [ ] Use PC-relative addressing
- [ ] Add tests for constant handling

## Milestone 6: Assembly Output

**Goal:** Generate assembly text output

### Tasks

- [ ] Create `pkg/asm/printer.go`
- [ ] Generate section directives:
  - [ ] `.text` for code
  - [ ] `.data` for initialized data
  - [ ] `.bss` for uninitialized data
- [ ] Generate function labels:
  - [ ] `.global f`
  - [ ] `.type f, @function`
  - [ ] `f:`
- [ ] Generate instructions:
  - [ ] Proper operand formatting
  - [ ] Comments for readability
- [ ] Generate data:
  - [ ] `.quad`, `.word`, `.byte`
  - [ ] String literals
- [ ] Handle alignment:
  - [ ] `.align` directives
- [ ] Add tests for assembly output

## Milestone 7: CLI Integration & Testing

**Goal:** Wire assembly generation to CLI, test against CompCert

### Tasks

- [ ] Add `-dasm` or `-S` flag implementation
- [ ] Output to `.s` file
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
