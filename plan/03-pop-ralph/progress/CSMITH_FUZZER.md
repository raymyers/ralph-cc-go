# Csmith Fuzzer Progress

## Goal
Set up automated fuzzing with csmith to find bugs in ralph-cc by comparing output against gcc.

## Status: COMPLETE

## Approach
1. Generate simple C programs using csmith (no stdio/complex features ralph-cc doesn't support)
2. Compile with both gcc and ralph-cc
3. Compare exit codes
4. Report mismatches with seed for reproduction

## Constraints Addressed
- ralph-cc doesn't support: hex literals, U/L suffixes, unary +, goto/labels
- csmith uses its own header - we extract only the relevant code
- Preprocessing converts hex to decimal, removes unsupported syntax

## Implementation
- Created `scripts/csmith-fuzz.sh` - headless fuzzing script
- Csmith options: `--no-pointers --no-arrays --no-structs --no-unions --no-bitfields --no-longlong --no-volatiles --no-argc --no-checksum --no-safe-math --no-jumps`
- Generates markdown report in `csmith-reports/`
- Categorizes failures: compilation failures, assembly failures, link failures, runtime crashes, output mismatches
- Preserves failing test cases with seeds for reproduction

## Results (initial run)
The fuzzer successfully found bugs:
- Many runtime crashes (segfaults, bus errors) 
- Output mismatches where gcc and ralph-cc produce different results
- Test cases preserved with reproducible seeds

## Usage
```bash
./scripts/csmith-fuzz.sh 100  # Run 100 iterations
cat csmith-reports/report-*.md  # View report
```

## Files Added
- `scripts/csmith-fuzz.sh` - Main fuzzing script
- `csmith-reports/` - Output directory (gitignored)
