# Function Pointer Typedef Parsing

## Status: COMPLETE

## Issue

Parser failed on function pointer typedefs like:
```c
typedef int (*callback_fn)(void*, int);
```

Error at `(`:
```
expected typedef name, got (
```

The parser's `parseTypedef()` expected an identifier after type+pointers, but function pointer typedefs have `(*name)(params)` pattern.

## Solution

Added `parseFunctionPointerTypedef()` method in `pkg/parser/parser.go`:
1. After parsing return type + stars, check if current token is `(` with peek `*`
2. If so, parse as function pointer typedef
3. Extract: return type, name, param types
4. Build type spec: `return_type(*)(param_types)`
5. Register typedef name

## Test Cases Added

In `testdata/integration.yaml`:
- Basic function pointer typedef: `typedef int (*callback)(void*, int);`
- Multiple params: `typedef int (*multi_callback)(void*, int, char**, char**);`

## Verification

```
$ echo 'typedef int (*callback)(void*, int); int main() { return 0; }' > /tmp/test.c
$ ./bin/ralph-cc --dparse /tmp/test.c
typedef int(*)(void*, int) callback;
int main() { return 0; }
```
