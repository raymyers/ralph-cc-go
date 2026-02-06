# Progress: Apply csmith fixes

## Task
Apply all fixes in `plan/05-fix-research-ralph/fixes` and confirm they fix the issue.

## Status: COMPLETE ✅

All three fixes have been applied and verified.

## Fixes Applied

### 1. Callee-Save Register Offsets (20260205-225448.md) ✅
**Issue**: Callee-saved registers saved at positive offsets from FP, which goes outside the stack frame.
**Root cause**: FP is set inside the frame, so callee-saves need negative offsets.
**Fix**:
- `pkg/stacking/calleesave.go`: Changed `ComputeCalleeSaveInfo` to use descending offsets (-8, -16, ...) from FP
- `pkg/stacking/layout.go`: Updated `CalleeSaveOffset` to -8 and `LocalOffset` to negative values
- `pkg/stacking/prolog.go`: Updated prologue/epilogue to use individual `SaveOffsets[i]` instead of `offset+8`
- Tests updated in `calleesave_test.go` and `layout_test.go`

### 2. Stack Slot Spill Handling (fail_compile_130805769.md) ✅
**Issue**: Stacking transform panics on stack slots in non-Lop instructions (Lload, Lstore, Lcond, Ljumptable, Lbuiltin).
**Fix**:
- `pkg/stacking/transform.go`: Added helper functions `ensureInReg` and `locsToRegsWithSpill`
- Added transform methods `transformLload`, `transformLstore`, `transformLcond`, `transformLjumptable`, `transformLbuiltin`
- These methods load spilled values from stack slots into temp registers before use

### 3. Paren Expression in Global Init (mismatch_263236830.md) ✅
**Issue**: `(-3)` not handled because `cabs.Paren` not unwrapped in `evaluateConstantInitializer`.
**Fix**:
- `pkg/clightgen/program.go`: Added `case cabs.Paren:` to recursively unwrap parenthesized expressions

## Verification
- `make test` - All unit tests pass
- `make check` - All tests including runtime tests pass
- Assembly output verified for correct callee-save offsets (now uses negative FP-relative addresses like `[x29, #-8]`)
