# Progress: run.sh Output Directory

## Task
Make `run.sh` build the final exe to an out/ dir and gitignore.

## Completed
1. Modified `scripts/run.sh`:
   - Added `OUT_DIR` variable pointing to `../out` relative to script
   - Created `out/` directory automatically with `mkdir -p`
   - Executables now placed in `out/` with basename of input file
2. Updated `.gitignore`:
   - Added `out/`
   - Removed specific binary entries (no longer needed)
3. Verified with `hello.c` and `fib.c` - both produce executables in `out/`
4. `make check` passes

## Status: Complete
