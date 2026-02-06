
# container-use

ALWAYS use ONLY Environments for ANY and ALL file, code, or shell operations—NO EXCEPTIONS—even for simple or generic requests.

DO NOT install or use the git cli with the environment_run_cmd tool. All environment tools will handle git operations for you. Changing ".git" yourself will compromise the integrity of your environment.

You MUST inform the user how to view your work using `container-use log <env_id>` AND `container-use checkout <env_id>`. Failure to do this will make your work inaccessible to others.

# ralph-cc

A prototype C compiler in Go, following CompCert's IR design for ARM64.

## Quick Build Commands

```bash
make build       # Build the compiler (~1s)
make test        # Fast tests (~2s), skips slow runtime tests
make test-slow   # Runtime tests only (~30s), requires as/ld
make check       # lint + test-all (can take 30s+, use sparingly)
```

**Important**: Use `make test` for quick iteration. Only run `make check` as final verification.

## Debugging the Compiler

Use IR dump flags to trace issues through compilation stages:

| Flag | Output | Description |
|------|--------|-------------|
| `--drtl` | RTL | High-level assembly-like, infinite pseudo-registers |
| `--dltl` | LTL | After register allocation, physical registers |
| `--dmach` | Mach | Concrete stack layout |
| `--dasm` | Assembly | Final ARM64 assembly |

Example:
```bash
./bin/ralph-cc --dasm testdata/example-c/fib.c
./bin/ralph-cc --drtl testdata/example-c/fib.c  # See before regalloc
```

### Debugging Flowchart

**Symptom → Which IR to inspect:**
- **Wrong output value** → `--drtl` (logic), then `--dltl` (register assignment)
- **Segfault at runtime** → `--dmach` (stack layout), `--dasm` (FP-relative addressing)
- **Register clobbered** → `--dltl` (callee-saved usage), `--dasm` (prologue/epilogue)

## Known Gotchas

1. **FP addressing**: All stack slots use FP-relative addressing, not SP
2. **Callee-saved registers**: X19-X28 must be used for values live across function calls
3. **Struct types**: Must resolve struct definitions when creating local variables
4. **Frame size**: Check Mach output for correct stack frame allocation
5. **Global variable types**: Global types must be registered in simplexpr's type environment before transforming function bodies, otherwise stores default to int32 size

## Bug Patterns

### Wrong memory access size for globals
**Symptom**: Global variable stores use `str` (32-bit) instead of `strb` (8-bit) or `strh` (16-bit)
**Cause**: Global variable types not registered in simplexpr type environment
**Check**: In Csharpminor output, verify stores use correct chunk (int8s, int16s, int32)
**Location**: `pkg/clightgen/program.go` - `translateFunctionWithStructsAndGlobals`

### Register allocation clobbers parameters (PARTIAL FIX)
**Symptom**: Functions with many parameters return wrong values; gcc works correctly
**Root Cause**: Parameters arrive in X0-X7 but local variable initialization can overwrite these registers before the parameter is used
**Attempted Fix**: Add conservative interference edges in `interference.go` between used parameters and all other pseudo-registers
**Current Status**: Fix is partial - helps some cases but doesn't fully solve the issue
**Debugging**: Compare `--dltl` output - check if parameter registers (X0-X7) are written before their last use
**Location**: `pkg/regalloc/interference.go` - see `BuildInterferenceGraph` parameter handling

## Test Data

Test files in `testdata/example-c/` cover:
- `hello.c` - Basic printf
- `fib.c`, `fib_fn.c` - Function calls, recursion
- `recursive.c` - Recursive factorial (callee-saved regs)
- `struct_point.c` - Struct field access
- `global_var.c` - Global variable access
- `many_args.c` - Stack argument passing (>8 args)
- `negative.c` - Signed arithmetic

## Key Documentation

- `docs/PHASES.md` - Compiler IR stages (RTL → LTL → Mach → Asm)
- `docs/TESTING.md` - Testing strategy and how to add tests
- `docs/RUNNING.md` - How to run compiled programs

# CompCert

CompCert is included as a submodule in the `compcert/` directory.

## Build Requirements

- OCaml 4.14+
- Coq 8.15-9.0 (we use 8.20.1)
- Menhir
- GCC (as assembler/linker)
- libgmp-dev, pkg-config

## Build Commands

```bash
# Install dependencies via opam
opam init --disable-sandboxing --auto-setup -y
eval $(opam env)
apt-get install -y libgmp-dev pkg-config
opam install -y coq.8.20.1 menhir

# Configure and build CompCert
cd compcert
./configure aarch64-linux -prefix /path/to/install
make -j$(nproc)
```

## Running the Compiler

```bash
./compcert/ccomp --version
./compcert/ccomp -c input.c -o output.o
```
