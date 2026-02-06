# Typedef with Leading Qualifiers

## Issue

Parser failed on `typedef const char *name;` with error:
```
expected type specifier in typedef, got const
```

The `const` keyword is a type qualifier, not a type specifier. The parser's `parseTypedef()` function checked for type specifiers first but didn't allow leading qualifiers.

## Fix

Modified `pkg/parser/parser.go` `parseTypedef()` to:
1. Collect leading type qualifiers (const, volatile, restrict) before checking for type specifier
2. Prepend those qualifiers to the final typeSpec string

## Files Changed

- `pkg/parser/parser.go` - Added qualifier handling in parseTypedef()
- `cmd/ralph-cc/integration_test.go` - Added test case
- `testdata/integration.yaml` - Added test case

## Verification

```bash
echo 'typedef const char *cstr;
cstr f() { return "hello"; }' > /tmp/test.c && ./bin/ralph-cc --dparse /tmp/test.c
```

Output:
```c
typedef const char* cstr;

char* f()
{
  return "hello";
}
```

## Next Blocker

Function pointer typedefs are not supported:
```c
typedef int (*callback)(void*, int);
```

This is task 3.2 (Handle function pointer types) in PLAN.md.
