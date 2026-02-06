# Inline Struct Definitions in Struct Fields

## Status: COMPLETE

## Problem

Parser failed on inline struct definitions within struct fields:

```c
struct outer {
  struct inner {   // Error: expected field name, got {
    int x;
  } *ptr;
};
```

This pattern appears frequently in SQLite (sqlite3_index_info, sqlite3_module, etc.).

## Solution Implemented

### Changes to `pkg/parser/parser.go`:

1. **Added fields to Parser struct**:
   - `inlineDefs []cabs.Definition` - collects inline struct/union definitions during parsing
   - `anonCounter int` - generates unique names for anonymous structs

2. **Modified `parseCompoundTypeSpecifier()`**:
   - When seeing `struct`/`union` followed by `{`, parse as inline definition
   - Generate anonymous name (`__anon_N`) if no tag name provided
   - Store definition in `p.inlineDefs` for later output
   - Return just the type reference (e.g., "struct inner")

3. **Added `parseInlineStructBody()`**:
   - Similar to `parseStructBody()` but doesn't consume trailing semicolon
   - Used for inline definitions within field declarations

4. **Modified `ParseProgram()`**:
   - Before adding each definition, prepend any collected `inlineDefs`
   - Reset `inlineDefs` after each top-level definition

### Output transformation:

Input:
```c
struct outer { struct inner { int x; } *ptr; };
```

Output:
```c
struct inner { int x; };  // Extracted and placed first
struct outer { struct inner* ptr; };  // References by name
```

## Tests Added

In `testdata/integration.yaml`:
- "inline struct in field (named)" 
- "inline struct in field (anonymous)"
- "multiple inline structs in field"

## Verified

- `make check` passes
- sqlite3_index_info pattern parses correctly
