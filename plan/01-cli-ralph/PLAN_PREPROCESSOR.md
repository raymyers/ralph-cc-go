# Phase: C Preprocessor

**Transformation:** C source → Preprocessed C source
**Prereqs:** Lexer, Parser (for macro expansion context)

A standalone C preprocessor replacing the current external `cc -E` dependency. This enables:
1. Full control over preprocessing behavior
2. Cross-platform consistency (no dependency on system compiler)
3. Better error messages and source location tracking
4. Foundation for future extensions (e.g., custom pragmas)

## Key References

| Resource | Purpose |
|----------|---------|
| C11 §6.10 | Preprocessing directives specification |
| C11 §6.10.1 | Conditional inclusion (#if, #ifdef, etc.) |
| C11 §6.10.2 | Source file inclusion (#include) |
| C11 §6.10.3 | Macro replacement (#define) |
| C11 §6.10.4 | Line control (#line) |
| C11 §6.10.5 | Error directive (#error) |
| C11 §6.10.6 | Pragma directive (#pragma) |
| [mcpp](https://sourceforge.net/projects/mcpp/) | Reference portable preprocessor |
| GCC cpp manual | Practical preprocessing behavior |

## Overview

The preprocessor operates on a token stream before parsing:
1. **Trigraph/digraph conversion** - (optional, low priority)
2. **Line splicing** - Join lines ending with `\`
3. **Tokenization** - Convert to preprocessing tokens
4. **Directive processing** - Handle `#` directives
5. **Macro expansion** - Expand `#define` macros
6. **Output** - Produce preprocessed source with `#line` markers

## Milestone 1: Preprocessor Lexer

**Goal:** Create a dedicated preprocessing lexer that handles raw source

**Status:** DONE

### Context

The preprocessing phase operates on "preprocessing tokens" which differ slightly from C tokens. We need a specialized lexer that:
- Handles `#` at the start of lines as directive markers
- Preserves whitespace information for macro expansion
- Supports line continuation (backslash-newline)
- Identifies preprocessing-specific tokens (e.g., `##` for token pasting)

### Tasks

- [x] Create `pkg/cpp/lexer.go` with preprocessing token types
- [x] Define preprocessing tokens:
  - [x] `PP_IDENTIFIER` - identifiers and keywords
  - [x] `PP_NUMBER` - preprocessing numbers (broader than C numbers)
  - [x] `PP_CHAR_CONST` - character constants
  - [x] `PP_STRING` - string literals
  - [x] `PP_PUNCTUATOR` - operators and punctuation
  - [x] `PP_HASH` - `#` at line start (directive marker)
  - [x] `PP_HASHHASH` - `##` (token pasting)
  - [x] `PP_NEWLINE` - significant for directive boundaries
  - [x] `PP_WHITESPACE` - preserved for macro spacing
- [x] Implement line continuation (`\` followed by newline)
- [x] Track source locations (file, line, column)
- [x] Handle comments (replace with single space per C spec)
- [x] Add tests for pp-token lexing

## Milestone 2: Include Path Resolution

**Goal:** Implement system and user include path searching

**Status:** DONE

### Context

`#include` has two forms:
- `#include <file>` - search system directories
- `#include "file"` - search current directory first, then system directories

We need to find system include directories automatically.

### Tasks

- [x] Create `pkg/cpp/include.go` for include path handling
- [x] Detect system include paths:
  - [x] Query `cc -v -E - </dev/null 2>&1` for include paths (bootstrap)
  - [x] Default paths: `/usr/include`, `/usr/local/include`
  - [x] macOS: `/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include`
  - [x] Linux: `/usr/include`, `/usr/lib/gcc/*/include`
- [x] Implement include search order:
  - [x] For `"file"`: current file's directory, then `-I` paths, then system
  - [x] For `<file>`: `-I` paths, then system
- [x] Handle `-I` flag to add include directories
- [x] Handle `-isystem` flag for system include directories
- [x] Detect and prevent circular includes
- [x] Add tests with mock file system

## Milestone 3: Directive Parser

**Goal:** Parse preprocessing directives

**Status:** DONE

### Context

Directives start with `#` at the beginning of a line (after whitespace). The directive parser extracts the directive name and arguments.

### Tasks

- [x] Create `pkg/cpp/directive.go` for directive parsing
- [x] Define directive types:
  - [x] `#include <file>` / `#include "file"`
  - [x] `#define NAME` / `#define NAME value` / `#define NAME(args) value`
  - [x] `#undef NAME`
  - [x] `#if expr`
  - [x] `#ifdef NAME` / `#ifndef NAME`
  - [x] `#elif expr`
  - [x] `#else`
  - [x] `#endif`
  - [x] `#line number ["filename"]`
  - [x] `#error message`
  - [x] `#warning message` (extension)
  - [x] `#pragma ...` (pass through or ignore)
  - [x] `# number "filename" [flags]` (GCC line marker)
- [x] Parse directive arguments correctly
- [x] Handle directive continuation lines (`\` at end) (via lexer)
- [x] Report errors for malformed directives
- [x] Add tests for directive parsing

## Milestone 4: Macro Definition and Storage

**Goal:** Implement macro definition storage and lookup

**Status:** DONE

### Context

Macros come in two forms:
- **Object-like**: `#define NAME value`
- **Function-like**: `#define NAME(a,b) ((a)+(b))`

Macros can be redefined only to identical definitions.

### Tasks

- [x] Create `pkg/cpp/macro.go` for macro handling
- [x] Define macro representation:
  - [x] Name
  - [x] Parameters (for function-like macros)
  - [x] Replacement list (as tokens)
  - [x] Variadic flag (for `...` parameter)
  - [x] Source location (for error messages)
- [x] Implement macro table (name → macro)
- [x] Handle built-in macros:
  - [x] `__FILE__` - current filename
  - [x] `__LINE__` - current line number
  - [x] `__DATE__` - compilation date
  - [x] `__TIME__` - compilation time
  - [x] `__STDC__` - always 1
  - [x] `__STDC_VERSION__` - 201112L for C11
- [x] Validate macro redefinitions (must be identical)
- [x] Implement `#undef`
- [x] Handle `-D` and `-U` command line flags
- [x] Add tests for macro storage

## Milestone 5: Macro Expansion

**Goal:** Implement macro expansion including argument substitution

**Status:** DONE

### Context

Macro expansion is the most complex part. Key rules:
- Macros are not expanded during their own expansion (prevent recursion)
- Function-like macro arguments are expanded before substitution
- `#` operator stringifies an argument
- `##` operator pastes tokens

### Tasks

- [x] Create `pkg/cpp/expand.go` for macro expansion
- [x] Implement object-like macro expansion
- [x] Implement function-like macro expansion:
  - [x] Parse argument list from invocation
  - [x] Expand arguments before substitution
  - [x] Handle variadic macros (`__VA_ARGS__`)
- [x] Implement stringification (`#arg`)
  - [x] Convert tokens to string with proper escaping
- [x] Implement token pasting (`a##b`)
  - [x] Concatenate adjacent tokens
  - [x] Handle invalid results (error)
- [x] Prevent recursive expansion (blue paint algorithm)
- [x] Handle empty arguments
- [x] Add comprehensive expansion tests

## Milestone 6: Conditional Compilation

**Goal:** Implement #if, #ifdef, #ifndef, #elif, #else, #endif

**Status:** TODO

### Context

Conditional compilation requires evaluating constant expressions. The expression can include:
- Integer constants
- `defined(NAME)` or `defined NAME`
- Arithmetic and logical operators
- Macro expansion (before evaluation)

### Tasks

- [ ] Create `pkg/cpp/conditional.go` for conditionals
- [ ] Implement condition evaluation:
  - [ ] Expand macros in expression first
  - [ ] Parse constant expression
  - [ ] Evaluate to integer result
  - [ ] Zero = false, non-zero = true
- [ ] Implement `defined` operator
- [ ] Handle `#ifdef NAME` as `#if defined(NAME)`
- [ ] Handle `#ifndef NAME` as `#if !defined(NAME)`
- [ ] Implement nested conditionals (stack-based)
- [ ] Skip tokens in false branches (still track nesting)
- [ ] Error on unbalanced conditionals
- [ ] Add tests for conditional compilation

## Milestone 7: Include Processing

**Goal:** Implement #include directive processing

**Status:** TODO

### Context

`#include` inserts the contents of another file. We need to:
- Search for the file
- Read and preprocess it
- Insert the result
- Generate proper `#line` directives

### Tasks

- [ ] Implement `#include "file"` (quoted form)
- [ ] Implement `#include <file>` (angle bracket form)
- [ ] Recursively preprocess included files
- [ ] Generate `#line` directives for file transitions
- [ ] Implement include guards recognition (optimization)
- [ ] Handle `#pragma once` (common extension)
- [ ] Track include stack for error messages
- [ ] Limit include depth to prevent stack overflow
- [ ] Add tests with multi-file includes

## Milestone 8: Output Generation

**Goal:** Generate preprocessed output with line markers

**Status:** TODO

### Context

Output should be suitable for our parser:
- Expanded source code
- `#line` directives for source mapping
- Preserved string literals and comments in strings

### Tasks

- [ ] Create `pkg/cpp/output.go` for output generation
- [ ] Generate `#line` directives at file boundaries
- [ ] Generate `#line` directives after macro expansion (optional)
- [ ] Preserve original line structure where possible
- [ ] Handle whitespace/newline preservation
- [ ] Support `-E` style output (for debugging)
- [ ] Support direct token stream output (for integration)
- [ ] Add output format tests

## Milestone 9: Main Preprocessor Driver

**Goal:** Tie all components together into a usable preprocessor

**Status:** TODO

### Context

The preprocessor driver coordinates all phases and provides the public API.

### Tasks

- [ ] Create `pkg/cpp/preprocess.go` as main driver
- [ ] Implement `Preprocess(filename, opts) (string, error)`
- [ ] Implement `PreprocessString(source, opts) (string, error)`
- [ ] Process directives in order:
  1. Read lines
  2. Handle line continuation
  3. Tokenize
  4. Process directives
  5. Expand macros
  6. Output result
- [ ] Integrate include path resolution
- [ ] Handle errors with good diagnostics
- [ ] Add integration tests

## Milestone 10: CLI Integration

**Goal:** Wire preprocessor into ralph-cc CLI

**Status:** TODO

### Context

Replace the external `cc -E` call with our internal preprocessor.

### Tasks

- [ ] Update `pkg/preproc/preproc.go` to use internal preprocessor
- [ ] Add `-E` flag to output preprocessed source only
- [ ] Add `-dpp` flag to debug preprocessor operation
- [ ] Maintain backward compatibility with external preprocessor (fallback?)
- [ ] Update `-I`, `-D`, `-U` flag handling
- [ ] Add `-isystem` flag support
- [ ] Test with existing test cases
- [ ] Test with system headers (stdio.h, stdlib.h, etc.)
- [ ] Update docs/INCLUDE.md with new capabilities

## Milestone 11: System Header Compatibility

**Goal:** Successfully preprocess standard library headers

**Status:** TODO

### Context

System headers use many extensions. We need to handle common ones:
- `__attribute__((...))`
- `__extension__`
- `__inline`, `__inline__`
- `_Pragma`
- `__has_include`
- `__has_feature` (clang)

### Tasks

- [ ] Define predefined macros for gcc/clang compatibility:
  - [ ] `__GNUC__`, `__GNUC_MINOR__`, `__GNUC_PATCHLEVEL__`
  - [ ] `__clang__` (if pretending to be clang)
  - [ ] `__SIZEOF_INT__`, `__SIZEOF_LONG__`, etc.
  - [ ] `__BYTE_ORDER__`, `__LITTLE_ENDIAN__`, `__BIG_ENDIAN__`
- [ ] Handle `__has_include(<file>)` / `__has_include("file")`
- [ ] Handle `__has_feature(x)` and `__has_extension(x)`
- [ ] Strip `__attribute__((...))` if parser doesn't support
- [ ] Handle `_Pragma("...")` operator
- [ ] Create test that successfully preprocesses:
  - [ ] `<stdio.h>`
  - [ ] `<stdlib.h>`
  - [ ] `<string.h>`
  - [ ] `<stdint.h>`
- [ ] Document any remaining limitations

## Testing Strategy

### Unit Tests

Each component should have focused unit tests:
- `pkg/cpp/lexer_test.go` - pp-token lexing
- `pkg/cpp/macro_test.go` - macro definition/expansion
- `pkg/cpp/conditional_test.go` - conditional compilation
- `pkg/cpp/include_test.go` - include resolution

### Integration Tests

Add to `testdata/preprocess.yaml`:
```yaml
- name: simple_define
  input: |
    #define X 42
    int a = X;
  expected: |
    int a = 42;

- name: function_macro
  input: |
    #define MAX(a,b) ((a)>(b)?(a):(b))
    int x = MAX(1, 2);
  expected: |
    int x = ((1)>(2)?(1):(2));
```

### Comparison Tests

Compare our output against system preprocessor:
- Preprocess with both, compare results
- Focus on semantic equivalence (ignore whitespace differences)

## Notes

### Scope Decisions

- **Trigraphs**: Not implementing (rarely used, deprecated in C23)
- **Digraphs**: Not implementing (rarely used)
- **UCN in identifiers**: Not implementing initially
- **`#embed`**: Not implementing (C23 feature)

### Performance Considerations

- Consider caching preprocessed headers (PCH-like)
- Memoize include guard detection
- Use efficient string interning for identifiers
