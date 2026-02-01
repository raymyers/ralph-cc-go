# CompCert Compilation Phases

CompCert transforms C source code through a series of intermediate representations (IRs), each with a formally verified semantics. Every transformation pass has a machine-checked proof that it preserves program behavior.

## Overview

```
C source
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                         FRONTEND                                │
├─────────────────────────────────────────────────────────────────┤
│  CompCert C ──► Clight ──► Csharpminor ──► Cminor ──► CminorSel │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                         BACKEND                                 │
├─────────────────────────────────────────────────────────────────┤
│  RTL ──► LTL ──► Linear ──► Mach ──► Asm                        │
│   │                                                             │
│   └── (multiple optimization passes at RTL level)               │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
Assembly
```

## Intermediate Languages

### Frontend Languages

| Language | Description |
|----------|-------------|
| **CompCert C** | Full C subset accepted by CompCert. Expressions may have side effects. |
| **Clight** | C without side-effects in expressions. Evaluation order is fixed. |
| **Csharpminor** | Low-level structured language. Type-dependent operations made explicit. |
| **Cminor** | Explicit stack allocation for address-taken local variables. |
| **CminorSel** | Cminor with target-specific operators and addressing modes. |

### Backend Languages

| Language | Description |
|----------|-------------|
| **RTL** | Register Transfer Language. CFG-based, infinite pseudo-registers, 3-address code. |
| **LTL** | Location Transfer Language. Physical registers + stack slots, basic blocks. |
| **Linear** | Linearized LTL with explicit labels and branches (no CFG). |
| **Mach** | Concrete activation record layout. Near-assembly abstraction. |
| **Asm** | Target assembly (ARM, PowerPC, RISC-V, x86). |

## Compilation Passes

### Frontend Passes

| Pass | Transformation | Source → Target |
|------|----------------|-----------------|
| **SimplExpr** | Pull side-effects out of expressions; fix evaluation order | CompCert C → Clight |
| **SimplLocals** | Pull non-addressable scalar locals out of memory | Clight → Clight |
| **Cshmgen** | Simplify control structures; make type-dependent operations explicit | Clight → Csharpminor |
| **Cminorgen** | Stack-allocate address-taken locals; simplify switch statements | Csharpminor → Cminor |
| **Selection** | Recognize target-specific operators and addressing modes; if-conversion | Cminor → CminorSel |
| **RTLgen** | Build CFG; generate 3-address code | CminorSel → RTL |

### RTL Optimization Passes

All optimizations operate on RTL and are optional (controlled by flags):

| Pass | Description | Flag |
|------|-------------|------|
| **Tailcall** | Recognize and optimize tail calls | `-ftailcalls` |
| **Inlining** | Inline function calls | `-finline` |
| **Renumber** | Postorder renumber CFG nodes | (internal) |
| **Constprop** | Global constant propagation | `-fconst-prop` |
| **CSE** | Common subexpression elimination | `-fcse` |
| **Deadcode** | Redundancy/dead code elimination | `-fredundancy` |
| **Unusedglob** | Remove unused static globals | (always) |

### Backend Passes

| Pass | Transformation | Source → Target |
|------|----------------|-----------------|
| **Allocation** | Register allocation (graph coloring with validation) | RTL → LTL |
| **Tunneling** | Branch tunneling (shortcut jumps to jumps) | LTL → LTL |
| **Linearize** | Linearize CFG into instruction sequence | LTL → Linear |
| **CleanupLabels** | Remove unreferenced labels | Linear → Linear |
| **Debugvar** | Synthesize debugging info for local variables | Linear → Linear |
| **Stacking** | Lay out activation records | Linear → Mach |
| **Asmgen** | Emit target assembly | Mach → Asm |

## Dumping Intermediate Representations

Use these flags to inspect IRs:

```bash
# Dump specific phases
ccomp -dparse input.c      # → input.parsed.c    (after parsing)
ccomp -dclight input.c     # → input.light.c     (Clight)
ccomp -drtl input.c        # → input.rtl.0-8     (RTL at each optimization)
ccomp -dltl input.c        # → input.ltl         (after register allocation)
ccomp -dmach input.c       # → input.mach        (Mach code)
ccomp -dasm input.c        # → input.s           (assembly)

# Dump all phases
ccomp -dparse -dclight -drtl -dltl -dmach -dasm -c input.c

# Post-linking validation info
ccomp -sdump input.c       # → input.json
```

### RTL Dump Points

The `-drtl` flag produces multiple files showing RTL at different optimization stages:

| File | After Pass |
|------|------------|
| `*.rtl.0` | Initial RTL (from RTLgen) |
| `*.rtl.1` | After tail call optimization |
| `*.rtl.2` | After inlining |
| `*.rtl.3` | After first renumbering |
| `*.rtl.4` | After constant propagation |
| `*.rtl.5` | After second renumbering |
| `*.rtl.6` | After CSE |
| `*.rtl.7` | After dead code elimination |
| `*.rtl.8` | After unused global removal |

## Semantic Preservation

Each pass has an associated proof module (e.g., `SimplExprproof.v`, `RTLgenproof.v`) proving semantic equivalence. The composition of all passes yields the main theorem:

> **Theorem**: If compilation succeeds, the generated assembly has the same observable behavior as the source C program.

"Observable behavior" includes:
- Termination/divergence
- Produced output (via external calls)
- Final return value

## Key Design Principles

1. **Small trusted computing base**: Only the Coq kernel and assembly semantics are trusted.

2. **Verified transformations**: Each pass proven correct, not just tested.

3. **Compositional proofs**: Individual pass proofs compose into whole-compiler correctness.

4. **Target independence**: Frontend and most optimizations are target-agnostic; only Selection, Stacking, and Asmgen are target-specific.

## References

- [Formal verification of a realistic compiler](https://xavierleroy.org/publi/compcert-CACM.pdf) - CACM 2009
- [A formally verified compiler back-end](https://xavierleroy.org/publi/compcert-backend.pdf) - JAR 2009
- [CompCert User Manual](https://compcert.org/man/)
