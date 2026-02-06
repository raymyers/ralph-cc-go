# Hex Literals Support

## Task
Fix lexer and parser to handle hex literals (`0x...`, `0X...`) and octal literals (`0...`) properly.

## Problem Found
The `readNumber()` function in `pkg/lexer/lexer.go` only handled decimal digits.
For input `0x09`, it read `0` and stopped. The `x09` became a separate identifier token.

Additionally, `parseIntegerLiteral()` in parser used `fmt.Sscanf("%d")` which only parses decimal.

## Test Case
```c
int x = 0x09;  // ERROR: expected ;, got IDENT (before fix)
```

## Fix Applied
1. Updated `readNumber()` in lexer to detect and read hex (`0x`/`0X`) and octal (`0...`) prefixes
2. Added `isHexDigit()` and `isOctalDigit()` helper functions
3. Handle integer suffixes (`u`, `U`, `l`, `L`, etc.) in lexer
4. Updated `parseIntegerLiteral()` in parser to:
   - Strip integer suffixes before parsing
   - Use `strconv.ParseInt(lit, 0, 64)` which auto-detects base
   - Fall back to `ParseUint` for large values

## Tests Added
- `TestHexAndOctalLiterals` in `pkg/lexer/lexer_test.go`
- Tests hex, octal, decimal, and suffix combinations

## Verification
```bash
# Test case now works:
# int x = 0x09;  -> x = 9
# int y = 0xFF;  -> y = 255
# int z = 0123;  -> z = 83 (octal)
make check  # all tests pass
```

## Status: COMPLETE
- [x] Identify root cause (lexer and parser)
- [x] Implement lexer fix
- [x] Implement parser fix
- [x] Add tests
- [x] Verify make check passes
