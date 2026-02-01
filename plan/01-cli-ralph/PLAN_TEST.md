# Test Plan

Gaps identified from spot-checking ralph-cc parser against CompCert ccomp -dparse.

## Parser Feature Gaps

These features parse correctly in CompCert but fail in ralph-cc:

[x] **struct/union types in function parameters** - `int getx(struct Point p)` fails
    - Error: "expected ')' after parameters, got IDENT"
    - Need to support `struct Name` as a type specifier in parameter lists
    - FIXED: Added struct/union/enum name handling in parseParameter()

[x] **C99 for-loop declarations** - `for (int i = 0; i < n; i++)` fails  
    - Error: "expected expression, got int"
    - Need to allow declaration as first part of for loop
    - FIXED: Added parseForDeclaration() and InitDecl field in For struct

[x] **Forward struct references in function signatures** - `struct Point *getp(struct Point *p)` fails
    - Error: "expected '{' or ';' after struct/union name, got *"
    - Need to support incomplete struct types (forward declarations) in return types and parameters
    - FIXED: Added peekPeekToken for lookahead, proper distinction between struct def vs function return type

## Output Format Differences (Cosmetic - Low Priority)

These are acceptable differences between ralph-cc and CompCert output:

- CompCert adds `extern void __builtin_debug(int kind, ...);` to all outputs
- CompCert converts `f()` to `f(void)` for empty parameter lists  
- CompCert adds implicit `return 0;` at end of main
- CompCert separates declarations from initializers (`int x; x = 1;` vs `int x = 1;`)
- CompCert adds `(int)` casts to switch case labels
- CompCert uses `int * p` spacing vs `int* p`

## Test Coverage Review

All parser commits were reviewed for test coverage:
- Every parser-related commit includes changes to `pkg/parser/parser_test.go`
- The YAML-driven test approach (`parse.yaml`) provides good parameterized coverage
- Integration tests in `testdata/integration.yaml` cover real-world scenarios

## Next Steps

1. Fix the parser feature gaps above (priority order)
2. Add corresponding test cases to `testdata/parse.yaml` and `pkg/parser/parser_test.go`
3. Re-run spot checks against CompCert to verify fixes
