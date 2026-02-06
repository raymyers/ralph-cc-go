# Progress: Apply csmith fixes

## Task
Apply all fixes in `plan/05-fix-research-ralph/fixes` and confirm they fix the issue.

## Fixes to Apply

### 1. Callee-Save Register Offsets (20260205-225448.md)
**Issue**: Callee-saved registers saved at positive offsets from FP, which goes outside the stack frame.
**Root cause**: FP is set inside the frame, so callee-saves need negative offsets.
**Status**: IN PROGRESS

### 2. Stack Slot Spill Handling (fail_compile_130805769.md)
**Issue**: Stacking transform panics on stack slots in non-Lop instructions.
**Status**: TODO

### 3. Paren Expression in Global Init (mismatch_263236830.md)
**Issue**: `(-3)` not handled because cabs.Paren not unwrapped.
**Status**: TODO

## Current Step
Analyzing callee-save offset fix. Need to understand frame layout first.

### Frame Layout Analysis

Looking at prolog.go, the prologue does:
1. `sub sp, sp, #TotalSize` 
2. `stp fp, lr, [sp, #TotalSize-16]` 
3. `add fp, sp, #TotalSize-16`

So with TotalSize=48:
- SP at bottom (0)
- FP = SP + 32 (points at saved FP/LR)
- Saved FP at SP+32, LR at SP+40

Currently callee-saves go at FP+16 = SP+48, which is OUTSIDE the frame!

Correct layout should place callee-saves BELOW FP (at negative offsets).
