# Progress: Systemic Common Causes

## Task
Address common causes systemically from `plan/05-fix-research-ralph/COMMON_CAUSES.md` at design, type or testing level.

## Status: COMPLETE ✅

## Common Causes Addressed

From COMMON_CAUSES.md:

### 1. Callee-Save Register Offset Sign Error ✅ (already fixed)
Applied in previous task.

### 2. Missing Paren Handling ✅ (already fixed)
Applied in previous task.

### 3. Stack Slot Handling ✅ (already fixed)
Applied in previous task.

### 4. Logical AND/OR Missing Short-Circuit ✅
**Location**: `pkg/simplexpr/transform.go`
**Fix**: Implemented `transformLogicalAnd()` and `transformLogicalOr()` functions that convert:
- `a && b` → `if (a) { if (b) temp=1 else temp=0 } else { temp=0 }`
- `a || b` → `if (a) { temp=1 } else { if (b) temp=1 else temp=0 }`

Also changed placeholders in `cabsToBinaryOp()` to panic instead of returning wrong bitwise operators, for early error detection.

Added tests in `testdata/e2e_runtime.yaml`:
- `C2.1 - logical and with nonzero values` (verifies -8 && -8 == 1, not -8)
- `C2.1 - logical or with nonzero values` (verifies -8 || 0 == 1)
- `C2.1 - logical and normalizes to 1` (verifies 5 && 10 == 1)
- `C2.1 - logical or normalizes to 1` (verifies 5 || 10 == 1)

### 5. Missing Typedef Resolution ✅
**Location**: `pkg/clightgen/types.go` and `pkg/simplexpr/transform.go`
**Fix**: Added support for standard integer typedefs from `<stdint.h>`:
- `int8_t`, `uint8_t`
- `int16_t`, `uint16_t`
- `int32_t`, `uint32_t`
- `int64_t`, `uint64_t`
- `size_t`, `ssize_t`, `ptrdiff_t`

Also added additional C type variants:
- `signed char`, `signed short`, `short int`, etc.
- `unsigned short`, `unsigned short int`

## Pre-existing Issues (not addressed)
- `C2.11 - array access` test was already failing before these changes (local array handling issue)
