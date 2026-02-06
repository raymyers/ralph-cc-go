# Cast Pointer Types

## Task
Fix cast expression parsing to support pointer types like `(char*)`, `(void*)`, `(const char*)`.

## Problem
Parser's `parseCast()` only consumed single-token type names:
```go
typeName := p.curToken.Literal
p.nextToken() // consume type name
if !p.curTokenIs(lexer.TokenRParen) {
    p.addError(...)  // Fails here for (char*) because next is * not )
```

This fails on:
- `(char*)ptr` - sees `*` instead of `)`
- `(void*)0` - same issue
- `(const char*)str` - const qualifier not consumed

## Fix Applied
Updated `parseCast()` in `pkg/parser/parser.go` to:
1. Skip leading type qualifiers (const, volatile, restrict)
2. Parse base type using `parseCompoundTypeSpecifier()` (handles struct, multi-word types)
3. Skip type qualifiers after base type (e.g., `char const *`)
4. Parse pointer markers (`*`) and append to type name
5. Skip qualifiers after pointers

Same fix applied to `parseSizeof()` for `sizeof(type*)` expressions.

## Verification
```c
void *p = (void*)0;           // Works
char *s = (char*)&x;          // Works  
const char *cs = (const char*)s;  // Works
return (int)p + (int)cs;      // Works
```

## Status: COMPLETE
- [x] Analyzed cast parsing code
- [x] Implemented fix for `parseCast()`
- [x] Implemented fix for `parseSizeof()`
- [x] Verified with test cases
