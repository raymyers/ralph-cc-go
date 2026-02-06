# Function Pointer Multi-Word Type Parameters

## Problem

Several related parsing issues blocking SQLite:

1. Multi-word types in function pointer parameters (`unsigned int` etc.) - `expected ), got int`
2. Post-type qualifiers (`char const *` syntax) - `expected ), got const`
3. Pointer-to-function-pointer parameters (`void (**fn)(int)`) - `expected ), got *`

## Root Causes

### Issue 1: Multi-word types
In `parseFunctionPointerParams()`, lines 1761-1763 only handle single-token types.

### Issue 2: Post-type qualifiers  
In `parseParameter()` and `parseFunctionPointerParams()`, no handling for qualifiers after base type.

### Issue 3: Pointer-to-function-pointer
In `parseFunctionPointerParameter()`, only consumed one `*` level.

## Fixes Applied

1. Use `parseCompoundTypeSpecifier()` for multi-word types
2. Add `for p.isTypeQualifier() { p.nextToken() }` after base type in all relevant functions
3. Count and handle multiple `*` levels in pointer-to-function-pointer patterns
4. Also fixed `parseFuncPtrParamTypes()` for struct fields with funcptr params

## Test Cases (all pass)

- `int foo(int(*)(unsigned int));`
- `int foo(char const **ptr);`
- `int foo(void (**pxFunc)(int));`
- `struct test { int (*fn)(void (**pxFunc)(int)); };`

## Status

COMPLETE - All fixes verified with `make check`. 

SQLite now parses past line 900 (was stuck at ~800). New blocker is nested/inline struct definitions (task 2.6).
