# Include Directive Support

ralph-cc has a fully integrated C preprocessor that handles `#include` directives, macro expansion, and conditional compilation. Unlike CompCert, ralph-cc does not require an external C compiler for preprocessing.

## How It Works

1. When compiling a `.c` file, ralph-cc runs the file through its built-in preprocessor (`pkg/cpp`)
2. The preprocessor expands all `#include` directives, macros, and conditional compilation
3. The preprocessed output (with `#line` directives) is then parsed by ralph-cc
4. The lexer handles `#line` directives to track source locations

## File Extensions

Following CompCert conventions:

| Extension | Preprocessed? |
|-----------|--------------|
| `.c`      | Yes - runs through preprocessor |
| `.i`      | No - assumed already preprocessed |
| `.p`      | No - assumed already preprocessed |

## Command Line Options

### Include Paths

Use `-I` to add directories to the user include search path:

```bash
ralph-cc -I./include -I/usr/local/include source.c -dparse
```

Use `--isystem` to add directories to the system include search path:

```bash
ralph-cc --isystem /opt/mylib/include source.c -dparse
```

### Macro Definitions

Use `-D` to define macros:

```bash
ralph-cc -D DEBUG source.c -dparse
ralph-cc -D MAX_SIZE=100 source.c -dparse
```

Use `-U` to undefine macros:

```bash
ralph-cc -U DEBUG source.c -dparse
```

### Preprocessor-Only Output

Use `-E` to output only the preprocessed source:

```bash
ralph-cc -E source.c > preprocessed.i
```

Use `-dpp` to debug preprocessing (writes `.i` file and outputs to stdout):

```bash
ralph-cc -dpp source.c
```

### External Preprocessor Fallback

Use `--external-cpp` to use the system's external preprocessor instead of the built-in one:

```bash
ralph-cc --external-cpp source.c -dparse
```

## Example Usage

### Basic Include

Create a header file `myheader.h`:
```c
#define MY_CONSTANT 42
int helper_function(int x);
```

Create a source file `main.c`:
```c
#include "myheader.h"

int main() {
    return MY_CONSTANT;
}
```

Compile with:
```bash
ralph-cc main.c -dparse
```

### System Includes

System headers like `<stdio.h>` are supported:

```c
#include <stdio.h>

int main() {
    return 0;
}
```

The preprocessor successfully handles major system headers including `stdio.h`, `stdlib.h`, `string.h`, and `stdint.h`. It supports GCC/Clang compatibility features needed for these headers:

- `__GNUC__`, `__GNUC_MINOR__`, `__GNUC_PATCHLEVEL__` macros
- `__has_include()`, `__has_feature()`, `__has_attribute()`, `__has_builtin()`
- Platform macros: `__APPLE__`, `__MACH__`, `__LP64__`, `__aarch64__`
- Type size and limit macros: `__SIZEOF_INT__`, `__INT_MAX__`, etc.

Note: While the preprocessor handles system headers, ralph-cc's parser may not support all constructs found in them (e.g., complex `__attribute__` syntax).

### Conditional Compilation

The preprocessor supports full conditional compilation:

```c
#define DEBUG 1

#ifdef DEBUG
int debug_level = 3;
#else
int debug_level = 0;
#endif

#if defined(__APPLE__) && defined(__aarch64__)
int platform = 1;  // Apple Silicon
#elif defined(__linux__)
int platform = 2;  // Linux
#else
int platform = 0;  // Other
#endif
```

### Macro Expansion

Both object-like and function-like macros are supported:

```c
#define PI 3.14159
#define MAX(a, b) ((a) > (b) ? (a) : (b))
#define STRINGIFY(x) #x
#define CONCAT(a, b) a##b

double area = PI * r * r;
int bigger = MAX(x, y);
const char *name = STRINGIFY(hello);  // "hello"
int CONCAT(var, 1) = 10;              // var1 = 10
```

## Predefined Macros

The preprocessor defines standard macros:

| Macro | Value |
|-------|-------|
| `__FILE__` | Current filename |
| `__LINE__` | Current line number |
| `__DATE__` | Compilation date |
| `__TIME__` | Compilation time |
| `__STDC__` | 1 |
| `__STDC_VERSION__` | 201112L (C11) |

Plus GCC/Clang compatibility macros for system header compatibility.

## Known Limitations

1. **`#include_next`**: Not supported (used rarely in system headers)

2. **`_Pragma` Operator**: Not yet implemented (the `#pragma` directive is supported)

3. **`__attribute__` Syntax**: Passed through but not parsed/stripped by the preprocessor. The parser must handle or ignore these.

4. **Trigraphs/Digraphs**: Not implemented (deprecated in C23)

## Implementation Details

The preprocessing is handled by:

- `pkg/cpp/` - Full preprocessor implementation
  - `lexer.go` - Preprocessing token lexer
  - `preprocess.go` - Main driver
  - `macro.go` - Macro definition and storage
  - `expand.go` - Macro expansion
  - `conditional.go` - `#if`/`#ifdef`/`#ifndef` handling
  - `include.go` - Include path resolution
  - `directive.go` - Directive parsing

- `pkg/preproc/preproc.go` - Integration layer
  - `Preprocess(filename, opts)` - Runs preprocessor on a file
  - `NeedsPreprocessing(filename)` - Checks if file needs preprocessing

## Comparison with CompCert

Unlike CompCert, ralph-cc:
- Has a built-in preprocessor (no external compiler needed)
- Supports all common preprocessing features (`-D`, `-U`, `-I`, `--isystem`)
- Can output preprocessed source with `-E`

Like CompCert, ralph-cc:
- Handles `#line` directives from preprocessor output
- Uses standard C11 preprocessing semantics
