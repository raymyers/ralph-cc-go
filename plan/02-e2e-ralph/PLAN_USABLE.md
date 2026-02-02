# Usability Assessment Plan

This document outlines a rigorous methodology to assess how close ralph-cc is to being a usable compiler for short programs (~100 lines) using common C features.

## Definition of "Usable"

A compiler is considered usable for ~100 line programs if it can:

1. **Parse** - Correctly parse common C syntax
2. **Compile** - Generate assembly for all supported constructs
3. **Run** - Produce executables that give correct results
4. **Integrate** - Work with system libraries (stdio, stdlib)

## Feature Categories

### Category 1: Core Features (Must Work)

These features are essential for any non-trivial C program:

| ID | Feature | Parser | Compile | Runtime | Notes |
|----|---------|--------|---------|---------|-------|
| C1.1 | Integer constants | [ ] | [ ] | [ ] | `42`, `0`, `-1` |
| C1.2 | Integer arithmetic | [ ] | [ ] | [ ] | `+`, `-`, `*`, `/`, `%` |
| C1.3 | Integer comparisons | [ ] | [ ] | [ ] | `<`, `<=`, `>`, `>=`, `==`, `!=` |
| C1.4 | Local variables | [ ] | [ ] | [ ] | `int x = 5;` |
| C1.5 | Assignment | [ ] | [ ] | [ ] | `x = y;` |
| C1.6 | Function definitions | [ ] | [ ] | [ ] | `int f(int x) { ... }` |
| C1.7 | Function calls | [ ] | [ ] | [ ] | `f(1, 2)` |
| C1.8 | Return statement | [ ] | [ ] | [ ] | `return x;` |
| C1.9 | If statement | [ ] | [ ] | [ ] | `if (x) ...` |
| C1.10 | If-else statement | [ ] | [ ] | [ ] | `if (x) ... else ...` |
| C1.11 | While loop | [ ] | [ ] | [ ] | `while (x) ...` |
| C1.12 | For loop | [ ] | [ ] | [ ] | `for (i=0; i<n; i++) ...` |

### Category 2: Extended Features (Should Work)

Features commonly used in practical programs:

| ID | Feature | Parser | Compile | Runtime | Notes |
|----|---------|--------|---------|---------|-------|
| C2.1 | Logical operators | [ ] | [ ] | [ ] | `&&`, `||`, `!` |
| C2.2 | Bitwise operators | [ ] | [ ] | [ ] | `&`, `|`, `^`, `~`, `<<`, `>>` |
| C2.3 | Increment/decrement | [ ] | [ ] | [ ] | `++x`, `x++`, `--x`, `x--` |
| C2.4 | Compound assignment | [ ] | [ ] | [ ] | `+=`, `-=`, `*=`, etc. |
| C2.5 | Ternary operator | [ ] | [ ] | [ ] | `x ? y : z` |
| C2.6 | Do-while loop | [ ] | [ ] | [ ] | `do { ... } while (x);` |
| C2.7 | Switch statement | [ ] | [ ] | [ ] | `switch (x) { case 1: ... }` |
| C2.8 | Break/continue | [ ] | [ ] | [ ] | `break;`, `continue;` |
| C2.9 | Pointers | [ ] | [ ] | [ ] | `int *p; *p = 5;` |
| C2.10 | Address-of | [ ] | [ ] | [ ] | `&x` |
| C2.11 | Arrays | [ ] | [ ] | [ ] | `int a[10]; a[0] = 1;` |
| C2.12 | String literals | [ ] | [ ] | [ ] | `"hello"` |
| C2.13 | Character literals | [ ] | [ ] | [ ] | `'x'`, `'\n'` |

### Category 3: Type System (For Practical Programs)

| ID | Feature | Parser | Compile | Runtime | Notes |
|----|---------|--------|---------|---------|-------|
| C3.1 | Char type | [ ] | [ ] | [ ] | `char c = 'x';` |
| C3.2 | Unsigned types | [ ] | [ ] | [ ] | `unsigned int x;` |
| C3.3 | Typedef | [ ] | [ ] | [ ] | `typedef int myint;` |
| C3.4 | Struct definition | [ ] | [ ] | [ ] | `struct Point { int x, y; };` |
| C3.5 | Struct member access | [ ] | [ ] | [ ] | `p.x`, `p->x` |
| C3.6 | Enum | [ ] | [ ] | [ ] | `enum Color { RED };` |
| C3.7 | Const qualifier | [ ] | [ ] | [ ] | `const int x = 5;` |
| C3.8 | Void type | [ ] | [ ] | [ ] | `void f() { }` |
| C3.9 | Pointer arithmetic | [ ] | [ ] | [ ] | `p + 1`, `p++` |
| C3.10 | Cast expressions | [ ] | [ ] | [ ] | `(int)x` |

### Category 4: I/O and Library Integration

| ID | Feature | Parser | Compile | Runtime | Notes |
|----|---------|--------|---------|---------|-------|
| C4.1 | Include stdio.h | [ ] | [ ] | [ ] | `#include <stdio.h>` |
| C4.2 | printf call | [ ] | [ ] | [ ] | `printf("hello\n");` |
| C4.3 | puts call | [ ] | [ ] | [ ] | `puts("hello");` |
| C4.4 | External functions | [ ] | [ ] | [ ] | `int printf(...);` |

## Test Methodology

### Phase 1: Create E2E Runtime Tests

Create a new test file `testdata/e2e_runtime.yaml` with test cases that:
1. Compile C source to assembly
2. Assemble and link using system tools
3. Run the executable
4. Verify the exit code matches expected value

Example test case format:
```yaml
tests:
  - name: "integer addition"
    input: |
      int main() { return 3 + 4; }
    expected_exit: 7

  - name: "while loop - sum 1 to 10"
    input: |
      int main() {
        int s = 0, n = 10;
        while (n > 0) { s = s + n; n = n - 1; }
        return s;
      }
    expected_exit: 55
```

### Phase 2: Run and Document Results

For each feature category:
1. Run all tests in that category
2. Mark Parser/Compile/Runtime status in the table above
3. Document specific failures with error messages

### Phase 3: Prioritize Fixes

Based on test results:
1. Identify critical blocking issues (Category 1 failures)
2. Create targeted bug fix tasks
3. Verify fixes with regression tests

## Success Criteria

The compiler is considered **minimally usable** when:
- 100% of Category 1 features pass all three stages (Parser/Compile/Runtime)
- 80% of Category 2 features pass all three stages
- hello.c with printf works correctly

## Current Status

**Assessment Date**: 2026-02-02

### Test Infrastructure

- [x] Created `testdata/e2e_runtime.yaml` with comprehensive test cases (60+ tests)
- [x] Added runtime test runner to `cmd/ralph-cc/integration_test.go` (TestE2ERuntimeYAML)
- [x] Tests compile C→assembly→object→executable and verify exit codes

### Results Summary

| Category | Subcategory | Status | Notes |
|----------|-------------|--------|-------|
| C1.1 | Integer constants | ✅ PASS | 0, 42, 255 all work |
| C1.2 | Integer arithmetic | ✅ PASS | +, -, *, /, % all work |
| C1.3 | Integer comparisons | ❌ FAIL | `<`, `>`, `==` compile as ADD not CMP |
| C1.4 | Local variables | ✅ PASS | Basic and multiple vars work |
| C1.5 | Assignment | ✅ PASS | Simple and chained work |
| C1.6 | Function definitions | ✅ PASS | With and without params |
| C1.7 | Function calls | ✅ PASS | Multiple args, nested calls |
| C1.8 | Return statement | ✅ PASS | Early return works |
| C1.9 | If statement | ❌ FAIL | Condition not evaluated (no CMP) |
| C1.10 | If-else statement | ❌ FAIL | Same issue - condition broken |
| C1.11 | While loop | ❌ FAIL | Condition broken, infinite loops |
| C1.12 | For loop | ❌ FAIL | Same condition issue |

### Critical Issues Found

#### Issue 1: Comparison Operators Compile as ADD

**Severity**: CRITICAL - Blocks all control flow

**Symptom**: `return 3 < 5;` compiles as:
```asm
mov w0, #3
mov w1, #5
add w0, w0, w1  ; Should be: cmp w0, w1; cset w0, lt
```

**Impact**: 
- All comparison expressions return wrong values
- All conditionals (`if`, `while`, `for`) cannot work
- Loops may run infinitely or not at all

**Root cause**: The code generation for comparison operations (`Olt`, `Ogt`, `Oeq`, etc.) 
appears to be generating the wrong operation. The operation selector is likely selecting 
an ADD operation instead of a compare-and-set operation.

#### Issue 2: Conditional Branches Without Flag Setting

**Severity**: CRITICAL - Related to Issue 1

**Symptom**: `if (0)` generates `b.gt` without preceding CMP instruction

**Impact**: Branch condition is undefined, leading to unpredictable behavior

### What Works (verified)

1. **Constants and arithmetic**: All basic math operations produce correct results
2. **Variables**: Local variable declaration, initialization, and assignment work
3. **Functions**: Definition, parameter passing, return values, and calls all work
4. **String literals**: "hello" style strings work with printf
5. **printf**: External function calls to libc work (hello.c runs correctly)

### What's Broken (verified)

1. **Comparisons as expressions**: `x < y`, `x == y` etc. don't produce 0/1
2. **Conditionals**: `if (condition)` doesn't evaluate condition correctly  
3. **Loops**: `while` and `for` loops with conditions don't terminate correctly
4. **Control flow**: Anything depending on boolean/comparison results

### Fix Tasks

[ ] **FIX-001**: Implement comparison code generation correctly
    - Location: pkg/selection/ or pkg/asmgen/
    - Fix Olt, Ogt, Oeq, One, Ole, Oge operations
    - Must generate: cmp + cset (for expressions) or cmp + b.cond (for control flow)

[ ] **FIX-002**: Ensure conditionals set comparison flags
    - if/while/for conditions must emit CMP before branch
    - Currently emitting b.gt without preceding CMP

### Usability Verdict

**NOT YET USABLE** for ~100 line programs with common features.

While basic arithmetic and function calls work correctly (which is impressive!), 
the broken comparison operations mean that any program requiring:
- Conditional logic (if/else)
- Loops (while/for with conditions)
- Boolean expressions

...will not work correctly.

**Estimated effort to reach "minimally usable"**: 
- 1-2 issues to fix (comparison codegen)
- Medium complexity - likely in instruction selection or assembly generation
- After fix: ~80% of Category 1 features should work

### Next Steps

1. [ ] Investigate comparison codegen in pkg/selection/ and pkg/asmgen/
2. [ ] Fix comparison operators to generate CMP + CSET/B.cond
3. [ ] Re-run test suite to verify fixes
4. [ ] Update feature matrix with final results
