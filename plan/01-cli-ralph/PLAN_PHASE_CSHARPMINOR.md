# Phase: Csharpminor Generation (Cshmgen)

**Transformation:** Clight → Csharpminor
**Prereqs:** Clight generation (PLAN_PHASE_CLIGHT.md)

Csharpminor is a low-level structured language where type-dependent operations are made explicit.

## Key CompCert Files to Study

| File | Purpose |
|------|---------|
| `cfrontend/Csharpminor.v` | Csharpminor AST definition |
| `cfrontend/Cshmgen.v` | Transformation from Clight |
| `cfrontend/Cshmgenproof.v` | Correctness proof (study for semantics) |
| `backend/Cminor.v` | Target language (similar structure) |

## Overview

Cshmgen transforms Clight to Csharpminor by:
1. **Making type-dependent operations explicit** - `+` becomes `addint`, `addfloat`, etc.
2. **Simplifying control structures** - Complex control flow normalized
3. **Introducing explicit memory operations** - Load/store with explicit sizes
4. **Removing implicit type conversions** - All casts explicit

## Milestone 1: Csharpminor AST Definition

**Goal:** Define the Csharpminor AST in Go

### Tasks

- [x] Create `pkg/csharpminor/ast.go` with node interfaces
- [x] Define Csharpminor constants:
  - [x] `Ointconst`, `Ofloatconst`, `Olongconst`, `Osingleconst`
- [x] Define Csharpminor expressions:
  - [x] `Evar` (global variable)
  - [x] `Etempvar` (local temporary)
  - [x] `Eaddrof` (address of global)
  - [x] `Econst` (constant)
  - [x] `Eunop` (typed unary operations)
  - [x] `Ebinop` (typed binary operations)
  - [x] `Eload` (explicit memory load with chunk)
- [x] Define memory chunks:
  - [x] `Mint8signed`, `Mint8unsigned`, `Mint16signed`, `Mint16unsigned`
  - [x] `Mint32`, `Mint64`, `Mfloat32`, `Mfloat64`, `Many32`, `Many64`
- [x] Define Csharpminor statements:
  - [x] `Sskip`, `Sset`, `Sstore`
  - [x] `Scall`, `Stailcall`
  - [x] `Sseq`, `Sifthenelse`
  - [x] `Sloop`, `Sblock`, `Sexit`
  - [x] `Sswitch`, `Sreturn`, `Slabel`, `Sgoto`
- [x] Define function and program structures
- [x] Add tests for AST construction

## Milestone 2: Operator Translation

**Goal:** Translate C operators to typed Csharpminor operators

### Tasks

- [x] Create `pkg/cshmgen/operators.go`
- [x] Map unary operators by type:
  - [x] `Onegint`, `Onegf`, `Onegl`, `Onegs` (negation)
  - [x] `Onotint`, `Onotl` (bitwise not)
  - [x] `Ocast8signed`, `Ocast8unsigned`, etc. (casts)
  - [x] `Osingleoffloat`, `Ofloatofsingle` (float conversions)
  - [x] `Ointoffloat`, `Ofloatofint`, etc. (int/float conversions)
- [x] Map binary operators by type:
  - [x] `Oadd`, `Osub`, `Omul`, `Odiv` (int variants)
  - [x] `Oaddf`, `Osubf`, `Omulf`, `Odivf` (float variants)
  - [x] `Oaddl`, `Osubl`, `Omull`, `Odivl` (long variants)
  - [x] `Oand`, `Oor`, `Oxor`, `Oshl`, `Oshr`, `Oshru` (bitwise)
  - [x] `Ocmp`, `Ocmpu`, `Ocmpf`, `Ocmpl` (comparisons)
- [x] Handle pointer arithmetic (add/sub with scaling)
- [x] Add tests for operator translation

## Milestone 3: Expression Translation

**Goal:** Translate Clight expressions to Csharpminor

### Tasks

- [x] Create `pkg/cshmgen/expr.go`
- [x] Translate simple expressions (variables, constants)
- [x] Translate unary expressions with type lookup
- [x] Translate binary expressions with type lookup
- [x] Translate memory access:
  - [x] Dereference → `Eload` with appropriate chunk
  - [x] Array subscript → address computation + `Eload` (handled via Ederef, as Clight desugars a[i] to *(a+i))
  - [x] Struct field → offset computation + `Eload`
- [x] Translate sizeof/alignof to constants
- [x] Handle address-of expressions
- [x] Add tests for expression translation

## Milestone 4: Statement Translation

**Goal:** Translate Clight statements to Csharpminor

### Tasks

- [ ] Create `pkg/cshmgen/stmt.go`
- [ ] Translate assignment:
  - [ ] Simple assignment → `Sstore`
  - [ ] Temporary assignment → `Sset`
- [ ] Translate control flow:
  - [ ] if/else → `Sifthenelse`
  - [ ] while/for → `Sloop` + `Sexit`
  - [ ] break → `Sexit n` (with nesting depth)
  - [ ] continue → appropriate `Sexit`
  - [ ] return → `Sreturn`
- [ ] Translate switch statements
- [ ] Translate function calls
- [ ] Handle blocks and sequencing
- [ ] Add tests for statement translation

## Milestone 5: CLI Integration & Testing

**Goal:** Wire Csharpminor generation to CLI

### Tasks

- [ ] Add `-dcsharpminor` flag implementation (note: CompCert doesn't have this flag, so this is optional)
- [ ] Create `pkg/csharpminor/printer.go` for debugging output
- [ ] Create test cases in `testdata/csharpminor/`
- [ ] Add integration tests
- [ ] Test full pipeline: C → Clight → Csharpminor

## Test Strategy

1. **Unit tests:** Operator and expression translation in isolation
2. **Integration tests:** Full transformation from Clight
3. **Type coverage:** Test all type combinations for operators
4. **Control flow:** Test all control flow transformations

## Notes

- Csharpminor makes memory access explicit with chunks
- All operations are typed (no implicit conversions)
- Exit statements use nesting depth for break/continue
- This is an internal IR - CompCert doesn't dump it directly

## Dependencies

- `pkg/clight` - Input AST (from PLAN_PHASE_CLIGHT.md)
- `pkg/ctypes` - Type information (from PLAN_PHASE_CLIGHT.md)
