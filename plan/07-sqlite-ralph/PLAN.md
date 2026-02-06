# SQLite Compilation Plan

Goal: Make ralph-cc compile SQLite (amalgamation) successfully, producing a working binary.

## Strategy

Use the SQLite amalgamation (single sqlite3.c file) as target. Work iteratively:
1. Run compiler, observe first error
2. Fix that error class
3. Repeat until preprocessing passes, then parsing, then codegen

Each milestone has verification command to know when complete.

---

## Milestone 1: Preprocessing ✅ COMPLETE

**Verification**: `./bin/ralph-cc -E checkouts/sqlite-amalgamation-3470200/sqlite3.c > /dev/null 2>&1 && echo PASS`

### Tasks

[x] 1.1 Fix preprocessor expression parsing - Current error: `#if: expected ')'` in system headers. The `#if` expression parser needs to handle complex expressions from macOS SDK headers. Progress: `progress/PREPROC_EXPR.md`

[x] 1.6 Fix multi-line macro arguments - Macro invocations spanning multiple lines failed with "unterminated macro argument list". Fixed by tracking paren depth in preprocessor. Also changed macro redefinition from error to warning for system header compatibility. Progress: `progress/MULTILINE_MACRO.md`

[~] 1.2 Handle `__has_feature`, `__has_extension`, `__has_builtin` - These clang-specific macros appear in system headers. **SKIPPED** - already handled, not blocking.

[~] 1.3 Handle `__attribute__` - System headers use `__attribute__((visibility(...)))` etc. **SKIPPED** - already handled, not blocking.

[~] 1.4 Handle `_Pragma` operator - Some headers use this for pragma handling. **SKIPPED** - not blocking.

[~] 1.5 Handle `typeof` / `__typeof__` - Used in some GNU extensions. **SKIPPED** - not blocking.

---

## Milestone 2: Parsing

**Verification**: `./bin/ralph-cc --dparse checkouts/sqlite-amalgamation-3470200/sqlite3.c 2>&1 | grep -q "parsed" && echo PASS`

### Tasks

[x] 2.0 Fix typedef with leading qualifiers - `typedef const char *name;` failed because parser didn't handle qualifiers before type specifiers. Progress: `progress/TYPEDEF_QUALIFIERS.md`

[x] 2.0b Fix function pointer typedef parsing - `typedef int (*callback)(void*, int);` failed with "expected typedef name" because parser expected identifier after type+pointers. Added `parseFunctionPointerTypedef()` to handle this pattern. Progress: `progress/FUNCPTR_TYPEDEF.md`

[x] 2.0c Fix function pointer returning function pointer - `void (*(*xDlSym)(params))(params)` in sqlite3_vfs struct failed to parse. Added `parseNestedFunctionPointerField()` to handle nested function pointer declarators. Progress: `progress/FP_RETURN_FP.md`

[x] 2.0d Fix funcptr parameter parsing issues - Multi-word types (`unsigned int`), post-type qualifiers (`char const *`), and pointer-to-function-pointer (`void (**fn)(int)`) in function pointer parameters. Progress: `progress/FUNCPTR_MULTIWORD.md`

[ ] 2.1 Audit C99/C11 features used by SQLite - Identify required features: designated initializers, compound literals, variadic macros, _Bool, etc. Progress: `progress/C_FEATURES.md`

[ ] 2.2 Implement designated initializers - `struct foo x = { .field = value }`. Progress: `progress/DESIGNATED_INIT.md`

[ ] 2.3 Implement compound literals - `(type){...}` expressions. Progress: `progress/COMPOUND_LIT.md`

[ ] 2.4 Implement `_Static_assert` / `static_assert` - C11 compile-time assertions. Progress: `progress/STATIC_ASSERT.md`

[ ] 2.5 Handle inline functions - `inline` and `static inline`. Progress: `progress/INLINE.md`

[x] 2.6 Handle inline struct/union definitions in fields - `struct { int x; } field;` and `struct name { ... } *ptr;`. Progress: `progress/INLINE_STRUCT_FIELD.md`

[x] 2.7 Fix hex and octal literal parsing - `0xFF`, `0123` failed to parse correctly. Lexer now handles prefixes, parser uses `strconv.ParseInt` with auto-detect base. Progress: `progress/HEX_LITERALS.md`

[x] 2.8 Fix cast expression parsing for pointer types - `(char*)`, `(void*)`, `(const char*)` failed. Updated `parseCast()` and `parseSizeof()` to handle full type syntax. Progress: `progress/CAST_POINTER.md`

[ ] 2.9 Handle flexible array members - `struct { int n; char data[]; }`. Progress: `progress/FLEX_ARRAY.md`

[ ] 2.10 Handle `restrict` keyword - C99 pointer qualifier. Progress: `progress/RESTRICT.md`

---

## Milestone 3: Type Checking

**Verification**: `./bin/ralph-cc --dclight checkouts/sqlite-amalgamation-3470200/sqlite3.c 2>&1 | grep -q "function" && echo PASS`

### Tasks

[ ] 3.1 Handle all integer types - `int64_t`, `uint32_t`, `size_t`, `intptr_t`, etc. Progress: `progress/INT_TYPES.md`

[ ] 3.2 Handle function pointer types - Complex function pointer declarations in SQLite. Progress: `progress/FUNC_PTR.md`

[ ] 3.3 Handle incomplete types - Forward-declared structs used as pointers. Progress: `progress/INCOMPLETE.md`

[ ] 3.4 Handle type qualifiers - `const`, `volatile`, `restrict` combinations. Progress: `progress/QUALIFIERS.md`

---

## Milestone 4: Code Generation

**Verification**: `./bin/ralph-cc --dasm checkouts/sqlite-amalgamation-3470200/sqlite3.c 2>&1 | head -20 | grep -q ".text" && echo PASS`

### Tasks

[ ] 4.1 Handle static/extern linkage - SQLite uses `static` extensively for internal functions. Progress: `progress/LINKAGE.md`

[ ] 4.2 Handle large switch statements - SQLite's SQL parser has large switch/case. Progress: `progress/SWITCH.md`

[ ] 4.3 Handle computed gotos (if used) - GNU extension `goto *ptr`. Progress: `progress/COMPUTED_GOTO.md`

[ ] 4.4 Handle varargs - `va_start`, `va_arg`, `va_end`. Progress: `progress/VARARGS.md`

[ ] 4.5 Handle large functions - Some SQLite functions are very large. Ensure no stack overflow in compiler. Progress: `progress/LARGE_FUNC.md`

---

## Milestone 5: Linking & Runtime

**Verification**: `./run.sh checkouts/sqlite-amalgamation-3470200/shell.c checkouts/sqlite-amalgamation-3470200/sqlite3.c && ./out/shell ":memory:" ".quit"`

### Tasks

[ ] 5.1 Link with system libraries - `-lm`, `-lpthread`, `-ldl` as needed. Progress: `progress/SYSTEM_LIBS.md`

[ ] 5.2 Handle external function declarations - Correctly extern libc functions. Progress: `progress/EXTERN_DECL.md`

[ ] 5.3 Verify basic operations - Create table, insert, select. Progress: `progress/BASIC_OPS.md`

---

## Notes

- SQLite amalgamation version: 3.47.2
- Location: `checkouts/sqlite-amalgamation-3470200/`
- Full sqlite repo also available at `checkouts/sqlite/` for reference
- Detailed investigation notes go in `plan/07-sqlite-ralph/notes/`
