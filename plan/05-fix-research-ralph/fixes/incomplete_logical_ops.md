# Fix: incomplete_logical_ops - Partial implementation breaks build

## Summary

The fix for `mismatch_2487828851` (logical AND/OR operators) was partially implemented. The switch cases in `transformBinary()` were added but the helper methods they call were never created, leaving the compiler unable to build.

## Evidence

Build error:
```
pkg/simplexpr/transform.go:385:12: t.transformLogicalAnd undefined (type *Transformer has no field or method transformLogicalAnd)
pkg/simplexpr/transform.go:389:12: t.transformLogicalOr undefined (type *Transformer has no field or method transformLogicalOr)
```

## Current State

In `pkg/simplexpr/transform.go`, lines 383-389 reference methods that don't exist:

```go
case cabs.OpAnd:
    // Logical && with short-circuit: a && b => a ? (b ? 1 : 0) : 0
    return t.transformLogicalAnd(expr.Left, expr.Right)  // MISSING!

case cabs.OpOr:
    // Logical || with short-circuit: a || b => a ? 1 : (b ? 1 : 0)
    return t.transformLogicalOr(expr.Left, expr.Right)   // MISSING!
```

## Fix Required

Complete the implementation by adding the `transformLogicalAnd` and `transformLogicalOr` methods. The full implementation is documented in `mismatch_2487828851.md`.

## Priority

**HIGH** - This is a build-breaking issue. The compiler cannot be compiled until this is resolved.

## Options

1. **Complete the fix**: Add the missing methods from `mismatch_2487828851.md` 
2. **Revert partial change**: Remove the case statements and revert to the previous behavior (bitwise ops, which is wrong but compiles)

Recommend option 1 - complete the fix properly.
