# Phase: RTL Optimizations (Optional)

**Transformation:** RTL → RTL
**Prereqs:** RTL generation (PLAN_PHASE_RTL.md)

RTL optimizations are optional passes that improve code quality. CompCert applies them in sequence, producing numbered dump files (`.rtl.0` through `.rtl.8`).

**Note:** These are optional for correctness but important for performance. Can be skipped initially and added later.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `backend/Tailcall.v` | Tail call optimization |
| `backend/Inlining.v` | Function inlining |
| `backend/Renumber.v` | CFG renumbering |
| `backend/Constprop.v` | Constant propagation |
| `backend/CSE.v` | Common subexpression elimination |
| `backend/Deadcode.v` | Dead code elimination |
| `backend/Unusedglob.v` | Remove unused globals |
| `backend/ValueAnalysis.v` | Value analysis (for Constprop) |

## RTL Dump Points

CompCert generates multiple RTL dumps:
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

## Pass 1: Tail Call Optimization

**Goal:** Convert eligible calls to tail calls
**Flag:** `-ftailcalls`

### Tasks

- [ ] Create `pkg/rtlopt/tailcall.go`
- [ ] Identify tail call candidates:
  - [ ] Call followed immediately by return
  - [ ] Return value matches call result (or void)
  - [ ] No intervening operations
- [ ] Check caller-save safety:
  - [ ] Callee doesn't use caller's stack frame
  - [ ] Arguments fit in registers
- [ ] Transform `Icall + Ireturn` → `Itailcall`
- [ ] Add tests for tail call optimization

## Pass 2: Inlining

**Goal:** Inline small functions
**Flag:** `-finline`

### Tasks

- [ ] Create `pkg/rtlopt/inlining.go`
- [ ] Identify inlining candidates:
  - [ ] Small functions (instruction count threshold)
  - [ ] Single-use functions
  - [ ] Always-inline hints
- [ ] Implement inlining:
  - [ ] Copy callee's CFG into caller
  - [ ] Rename registers (fresh names)
  - [ ] Connect entry/exit points
  - [ ] Replace call with inlined body
- [ ] Handle recursive functions (don't inline)
- [ ] Add tests for inlining

## Pass 3: Renumber

**Goal:** Clean up CFG node numbering
**Internal pass**

### Tasks

- [ ] Create `pkg/rtlopt/renumber.go`
- [ ] Postorder traversal of CFG
- [ ] Assign sequential node numbers
- [ ] Update all references
- [ ] Benefits: better code layout, simplifies other passes
- [ ] Add tests for renumbering

## Pass 4: Constant Propagation

**Goal:** Propagate and fold constants
**Flag:** `-fconst-prop`

### Tasks

- [ ] Create `pkg/rtlopt/constprop.go`
- [ ] Implement value analysis:
  - [ ] Track known constant values
  - [ ] Track known ranges
  - [ ] Propagate through operations
- [ ] Constant folding:
  - [ ] `add(5, 3)` → `8`
  - [ ] `x * 0` → `0`
  - [ ] `x + 0` → `x`
- [ ] Algebraic simplification:
  - [ ] `x - x` → `0`
  - [ ] `x & 0` → `0`
- [ ] Branch simplification:
  - [ ] `if (true) A else B` → `A`
- [ ] Add tests for constant propagation

## Pass 5: Common Subexpression Elimination

**Goal:** Reuse previously computed values
**Flag:** `-fcse`

### Tasks

- [ ] Create `pkg/rtlopt/cse.go`
- [ ] Build available expression sets:
  - [ ] Dataflow analysis
  - [ ] Kill on store, call, etc.
- [ ] Identify redundant computations:
  - [ ] Same operation with same inputs
  - [ ] Result still available (not killed)
- [ ] Replace redundant computation:
  - [ ] Use previously computed value
  - [ ] May need to extend live range
- [ ] Add tests for CSE

## Pass 6: Dead Code Elimination

**Goal:** Remove unused computations
**Flag:** `-fredundancy`

### Tasks

- [ ] Create `pkg/rtlopt/deadcode.go`
- [ ] Compute liveness:
  - [ ] Backward dataflow
  - [ ] Live if used later or has side effects
- [ ] Identify dead instructions:
  - [ ] Result never used
  - [ ] No side effects
- [ ] Remove dead instructions:
  - [ ] Replace with Inop or remove node
- [ ] Remove unreachable code:
  - [ ] Nodes not reachable from entry
- [ ] Add tests for dead code elimination

## Pass 7: Unused Global Removal

**Goal:** Remove unused static globals
**Always runs**

### Tasks

- [ ] Create `pkg/rtlopt/unusedglob.go`
- [ ] Identify used globals:
  - [ ] Referenced in any function
  - [ ] External linkage (keep)
- [ ] Remove unused static globals:
  - [ ] Internal linkage only
  - [ ] Not referenced anywhere
- [ ] Add tests for unused global removal

## CLI Integration

### Tasks

- [ ] Honor optimization flags:
  - [ ] `-ftailcalls` (default: on with -O)
  - [ ] `-finline` (default: on with -O)
  - [ ] `-fconst-prop` (default: on with -O)
  - [ ] `-fcse` (default: on with -O)
  - [ ] `-fredundancy` (default: on with -O)
- [ ] `-O0` disables all optimizations
- [ ] `-O` (or `-O1`) enables basic optimizations
- [ ] Generate numbered RTL dumps with `-drtl`:
  - [ ] `.rtl.0` after RTLgen
  - [ ] `.rtl.1` through `.rtl.8` after each pass
- [ ] Test against CompCert dumps

## Test Strategy

1. **Unit tests:** Each optimization in isolation
2. **Correctness:** Output functionally equivalent to input
3. **Effectiveness:** Measure optimization impact
4. **Regression:** Ensure optimizations don't break code
5. **Golden tests:** Match CompCert's optimized RTL

## Implementation Order

Recommended order for implementation:
1. **Renumber** - Simplest, useful for other passes
2. **Dead code** - Simple liveness analysis
3. **Constant propagation** - Medium complexity
4. **CSE** - Requires available expressions
5. **Unused globals** - Simple global analysis
6. **Tail calls** - Requires call site analysis
7. **Inlining** - Most complex

## Notes

- All optimizations are optional for correctness
- CompCert proves each optimization preserves semantics
- We can start with no optimizations and add incrementally
- Running with `-O0` skips all (good for debugging)

## Dependencies

- `pkg/rtl` - RTL AST (from PLAN_PHASE_RTL.md)
- Dataflow analysis framework (shared infrastructure)
