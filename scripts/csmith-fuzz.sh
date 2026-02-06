#!/bin/bash
# csmith-fuzz.sh - Automated fuzzing with csmith to find compiler bugs
#
# Usage: ./scripts/csmith-fuzz.sh [iterations]
#
# Compares ralph-cc output against gcc on randomly generated C programs.
# Reports mismatches with reproducible seeds.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RALPH_CC="$PROJECT_DIR/bin/ralph-cc"
REPORT_DIR="$PROJECT_DIR/csmith-reports"
WORK_DIR="$REPORT_DIR/work"
ITERATIONS=${1:-100}

# Csmith include path
CSMITH_INCLUDE=$(dirname $(dirname $(which csmith)))/include/csmith-2.3.0

mkdir -p "$REPORT_DIR" "$WORK_DIR"

# Initialize report
REPORT_FILE="$REPORT_DIR/report-$(date +%Y%m%d-%H%M%S).md"
cat > "$REPORT_FILE" << EOF
# Csmith Fuzzing Report

Date: $(date)
Iterations: $ITERATIONS

## Failures

EOF

# Counters
total=0
passed=0
ralph_fail=0
mismatch=0
gcc_skip=0

echo "Starting csmith fuzzing with $ITERATIONS iterations..."
echo "Report: $REPORT_FILE"
echo ""

for i in $(seq 1 $ITERATIONS); do
    seed=$RANDOM$RANDOM
    csmith_file="$WORK_DIR/csmith_$seed.c"
    test_file="$WORK_DIR/test_$seed.c"
    
    # Generate simple C program without features ralph-cc doesn't support
    csmith --seed $seed \
        --no-pointers \
        --no-arrays \
        --no-structs \
        --no-unions \
        --no-bitfields \
        --no-longlong \
        --no-volatiles \
        --no-argc \
        --no-checksum \
        --no-safe-math \
        --no-jumps \
        --max-funcs 3 \
        --max-block-depth 3 \
        --max-expr-complexity 5 \
        > "$csmith_file" 2>/dev/null
    
    # Create a standalone C file without csmith.h dependency
    # Extract global variables and add typedefs
    cat > "$test_file" << 'HEADER'
// Standalone csmith output
typedef signed char int8_t;
typedef unsigned char uint8_t;
typedef short int16_t;
typedef unsigned short uint16_t;
typedef int int32_t;
typedef unsigned int uint32_t;

HEADER

    # Extract static global variables (the g_N pattern)
    grep -E '^static (u?int(8|16|32)_t|long|int|unsigned|short|char) g_[0-9]+' "$csmith_file" >> "$test_file" 2>/dev/null || true
    echo "" >> "$test_file"
    
    # Extract function forward declarations
    grep -E '^static .* func_[0-9]+\(.*\);' "$csmith_file" >> "$test_file" 2>/dev/null || true
    echo "" >> "$test_file"
    
    # Extract function definitions - everything from first function def to int main
    # Use awk to extract function bodies
    awk '
        /^static .* func_[0-9]+\(/ { in_func=1 }
        /^int main / { in_func=0 }
        in_func { print }
    ' "$csmith_file" >> "$test_file"
    
    # Convert hex literals to decimal (ralph-cc doesn't support hex yet)
    # Also remove U/L suffixes and unary + operator
    perl -i -pe '
        s/0x([0-9A-Fa-f]+)[UuLl]*/hex($1)/ge;  # hex to decimal
        s/(\d+)[UuLl]+/$1/g;                   # remove U/L suffixes
        s/\(\+(\w)/\($1/g;                     # remove unary + before word (var/func)
        s/\(\+\(/\(\(/g;                       # remove unary + before paren
    ' "$test_file"
    
    # Find the first global variable name for our return value
    first_global=$(grep -oE 'g_[0-9]+' "$test_file" | head -1)
    if [ -z "$first_global" ]; then
        first_global="0"
    fi
    
    # Add simple main that calls func_1 and returns a computed value
    # Use 127 instead of 0x7F since ralph-cc doesn't support hex yet
    cat >> "$test_file" << EOF

int main(void) {
    func_1();
    return (int)($first_global) & 127;
}
EOF

    total=$((total + 1))
    
    # Try to compile with gcc first
    gcc_out="$WORK_DIR/gcc_$seed"
    if ! gcc -O0 -w -o "$gcc_out" "$test_file" 2>/dev/null; then
        # Skip if gcc can't compile (extraction failed)
        gcc_skip=$((gcc_skip + 1))
        rm -f "$csmith_file" "$test_file"
        continue
    fi
    
    # Get gcc result
    gcc_exit=0
    "$gcc_out" >/dev/null 2>&1 || gcc_exit=$?
    
    # Try to compile with ralph-cc
    ralph_asm="$WORK_DIR/test_$seed.s"
    if ! "$RALPH_CC" -dasm "$test_file" >/dev/null 2>&1; then
        ralph_fail=$((ralph_fail + 1))
        echo "[$i/$ITERATIONS] RALPH_FAIL seed=$seed"
        echo "### seed $seed: ralph-cc compilation failed" >> "$REPORT_FILE"
        echo '```c' >> "$REPORT_FILE"
        cat "$test_file" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        cp "$test_file" "$REPORT_DIR/fail_compile_$seed.c"
        rm -f "$csmith_file" "$gcc_out"
        continue
    fi
    
    # Try to assemble and link with run.sh approach
    ralph_out="$WORK_DIR/ralph_$seed"
    macos_asm="$WORK_DIR/test_${seed}_macos.s"
    obj_file="$WORK_DIR/test_$seed.o"
    
    # Convert to macOS format (same as run.sh)
    perl -ne '
        BEGIN { $adrp_label = ""; }
        s/^\s*\.type.*\n//;
        s/^\s*\.size.*\n//;
        s/^\s*\.section\s+\.rodata.*/.section __DATA,__const/;
        s/\.global\s+([a-zA-Z_][a-zA-Z0-9_]*)/.global _\1/;
        s/^([a-zA-Z_][a-zA-Z0-9_]*):/_\1:/;
        s/\bbl\s+([a-zA-Z_][a-zA-Z0-9_]*)/bl _\1/;
        if (/\badrp\s+(\w+),\s*(\.L\w+)/) {
            $adrp_label = $2;
            s/\badrp\s+(\w+),\s*(\.L\w+)/adrp\t$1, $2\@PAGE/;
        }
        elsif (/\badrp\s+(\w+),\s*([a-zA-Z_][a-zA-Z0-9_]*)/) {
            $adrp_label = "_$2";
            s/\badrp\s+(\w+),\s*([a-zA-Z_][a-zA-Z0-9_]*)/adrp\t$1, _$2\@PAGE/;
        }
        elsif ($adrp_label ne "" && /^\s*add\s+(\w+),\s*(\w+),\s*#0\s*\n/) {
            s/^\s*add\s+(\w+),\s*(\w+),\s*#0\s*\n/\tadd\t$1, $2, $adrp_label\@PAGEOFF\n/;
            $adrp_label = "";
        }
        else {
            $adrp_label = "";
        }
        print;
    ' "$ralph_asm" > "$macos_asm"
    
    if ! as -o "$obj_file" "$macos_asm" 2>/dev/null; then
        ralph_fail=$((ralph_fail + 1))
        echo "[$i/$ITERATIONS] ASM_FAIL seed=$seed"
        echo "### seed $seed: assembly failed" >> "$REPORT_FILE"
        cp "$test_file" "$REPORT_DIR/fail_asm_$seed.c"
        cp "$ralph_asm" "$REPORT_DIR/fail_asm_$seed.s"
        rm -f "$csmith_file" "$gcc_out" "$macos_asm"
        continue
    fi
    
    SDK_PATH=$(xcrun --show-sdk-path)
    if ! ld -o "$ralph_out" "$obj_file" -lSystem -L"$SDK_PATH/usr/lib" 2>/dev/null; then
        ralph_fail=$((ralph_fail + 1))
        echo "[$i/$ITERATIONS] LINK_FAIL seed=$seed"
        echo "### seed $seed: linking failed" >> "$REPORT_FILE"
        cp "$test_file" "$REPORT_DIR/fail_link_$seed.c"
        rm -f "$csmith_file" "$gcc_out" "$macos_asm" "$obj_file"
        continue
    fi
    
    # Get ralph-cc result
    ralph_exit=0
    "$ralph_out" >/dev/null 2>&1 || ralph_exit=$?
    
    # Check for crashes (exit code > 128 indicates signal)
    if [ "$ralph_exit" -gt 128 ]; then
        ralph_fail=$((ralph_fail + 1))
        sig=$((ralph_exit - 128))
        echo "[$i/$ITERATIONS] CRASH seed=$seed signal=$sig"
        echo "### seed $seed: runtime crash (signal $sig)" >> "$REPORT_FILE"
        echo '```c' >> "$REPORT_FILE"
        cat "$test_file" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        cp "$test_file" "$REPORT_DIR/crash_$seed.c"
        cp "$ralph_asm" "$REPORT_DIR/crash_$seed.s"
    # Compare exit codes
    elif [ "$gcc_exit" -eq "$ralph_exit" ]; then
        passed=$((passed + 1))
        printf "[$i/$ITERATIONS] PASS seed=$seed exit=$gcc_exit\r"
    else
        mismatch=$((mismatch + 1))
        echo "[$i/$ITERATIONS] MISMATCH seed=$seed gcc=$gcc_exit ralph=$ralph_exit"
        echo "### seed $seed: output mismatch" >> "$REPORT_FILE"
        echo "- gcc exit: $gcc_exit" >> "$REPORT_FILE"
        echo "- ralph exit: $ralph_exit" >> "$REPORT_FILE"
        echo '```c' >> "$REPORT_FILE"
        cat "$test_file" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        cp "$test_file" "$REPORT_DIR/mismatch_$seed.c"
        cp "$ralph_asm" "$REPORT_DIR/mismatch_$seed.s"
    fi
    
    # Cleanup work files
    rm -f "$csmith_file" "$test_file" "$ralph_asm" "$macos_asm" "$obj_file" "$gcc_out" "$ralph_out"
done

echo ""
echo ""

# Calculate pass rate safely
if [ "$total" -gt "$gcc_skip" ]; then
    valid=$((total - gcc_skip))
    pass_rate=$(echo "scale=1; $passed * 100 / $valid" | bc)
else
    valid=0
    pass_rate="N/A"
fi

# Write summary
cat >> "$REPORT_FILE" << EOF

## Results

| Metric | Count |
|--------|-------|
| Total iterations | $total |
| GCC skipped (extraction failed) | $gcc_skip |
| Valid tests | $valid |
| Passed | $passed |
| Ralph-cc failures | $ralph_fail |
| Output mismatches | $mismatch |

Pass rate: ${pass_rate}%
EOF

echo "=== Fuzzing Complete ==="
echo "Total: $total"
echo "GCC skipped: $gcc_skip"
echo "Valid: $valid"
echo "Passed: $passed"
echo "Ralph-cc failures: $ralph_fail"  
echo "Output mismatches: $mismatch"
echo ""
echo "Report: $REPORT_FILE"

# Cleanup work dir
rm -rf "$WORK_DIR"
