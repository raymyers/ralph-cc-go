# Hello.c Verification Status

## Status: WORKING ✓

As of this verification, `testdata/example-c/hello.c` compiles and runs correctly:

```bash
$ ./scripts/run.sh testdata/example-c/hello.c
==> Compiling testdata/example-c/hello.c to ARM64 assembly...
==> Converting to macOS format...
==> Assembling...
==> Linking...
==> Running testdata/example-c/hello...
---
Hello, World!
---
Exit code: 0
```

## Issues Fixed

### 1. External Function Calls Generated Indirect Calls

**Symptom**: `printf()` was being called with `blr x0` (indirect call through register) instead of `bl printf` (direct call to symbol).

**Root Cause**: The selection phase (`pkg/selection/stmt.go`) didn't add external function names to the `Globals` map. Only functions defined in the program were added. When `SelectExpr` processed `Evar{Name: "printf"}`, it wasn't recognized as a global symbol and was treated as a local variable.

**Fix**: Added `collectExternalFunctions()` function that scans the program for calls to undefined functions and adds them to the globals set before selection.

### 2. ADRP/ADD Relocation on macOS

**Symptom**: macOS assembler rejected `adrp x0, .Lstr0` with "ADR/ADRP relocations must be GOT relative".

**Root Cause**: macOS requires `@PAGE` and `@PAGEOFF` suffixes for ADRP-based address loading of local labels.

**Fix**: Updated `scripts/run.sh` to transform:
```asm
adrp    x0, .Lstr0
add     x0, x0, #0
```
into:
```asm
adrp    x0, .Lstr0@PAGE
add     x0, x0, .Lstr0@PAGEOFF
```

## Tests Added

1. **E2E Test**: `testdata/e2e_asm.yaml` - "external function call" test case
2. **Unit Tests** in `pkg/selection/stmt_test.go`:
   - `TestSelectProgram_ExternalFunctions`
   - `TestCollectExternalFunctions`
   - `TestCollectExternalFunctions_Nested`

## Remaining Notes

The `__sputc` inline function from `<stdio.h>` is still being compiled into the output. This is harmless but could be cleaned up in the future by detecting inline functions that aren't actually called.