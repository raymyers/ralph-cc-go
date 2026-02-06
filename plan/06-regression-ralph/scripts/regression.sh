#!/bin/bash
# regression.sh - Verify that previously found csmith bugs stay fixed
#
# Usage: ./plan/06-regression-ralph/scripts/regression.sh
#
# Regenerates test cases from known seeds and verifies ralph-cc produces
# correct output compared to gcc. All seeds are embedded in this script
# to avoid relying on the csmith-reports folder.

set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
RALPH_CC="$PROJECT_DIR/bin/ralph-cc"
WORK_DIR=$(mktemp -d)

trap 'rm -rf "$WORK_DIR"' EXIT

# All known seeds from csmith-reports findings
# Format: TYPE:SEED where TYPE is crash, mismatch, or fail_compile
KNOWN_ISSUES=(
    "crash:102911892"
    "crash:1093823871"
    "crash:114128045"
    "crash:1176020246"
    "crash:145691413"
    "crash:184567722"
    "crash:1870324845"
    "crash:1950716464"
    "crash:23463107"
    "crash:2690612573"
    "crash:3051214201"
    "crash:3253220824"
    "crash:55976753"
    "crash:785411410"
    "crash:853828320"
    "crash:928313049"
    "fail_compile:130805769"
    "mismatch:2487828851"
    "mismatch:263236830"
)

# Generate a csmith test file from seed (same logic as csmith-fuzz.sh)
generate_test() {
    local seed=$1
    local output=$2
    local csmith_file="$WORK_DIR/csmith_$seed.c"
    
    # Generate simple C program without features ralph-cc doesn't support
    csmith --seed "$seed" \
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
    cat > "$output" << 'HEADER'
// Standalone csmith output
typedef signed char int8_t;
typedef unsigned char uint8_t;
typedef short int16_t;
typedef unsigned short uint16_t;
typedef int int32_t;
typedef unsigned int uint32_t;

HEADER

    # Extract static global variables (the g_N pattern)
    grep -E '^static (u?int(8|16|32)_t|long|int|unsigned|short|char) g_[0-9]+' "$csmith_file" >> "$output" 2>/dev/null || true
    echo "" >> "$output"

    # Extract function forward declarations
    grep -E '^static .* func_[0-9]+\(.*\);' "$csmith_file" >> "$output" 2>/dev/null || true
    echo "" >> "$output"

    # Extract function definitions - everything from first function def to int main
    awk '
        /^static .* func_[0-9]+\(/ { in_func=1 }
        /^int main / { in_func=0 }
        in_func { print }
    ' "$csmith_file" >> "$output"

    # Convert hex literals to decimal (ralph-cc doesn't support hex yet)
    # Also remove U/L suffixes and unary + operator
    perl -i -pe '
        s/0x([0-9A-Fa-f]+)[UuLl]*/hex($1)/ge;  # hex to decimal
        s/(\d+)[UuLl]+/$1/g;                   # remove U/L suffixes
        s/\(\+(\w)/\($1/g;                     # remove unary + before word (var/func)
        s/\(\+\(/\(\(/g;                       # remove unary + before paren
    ' "$output"

    # Find the first global variable name for our return value
    local first_global
    first_global=$(grep -oE 'g_[0-9]+' "$output" | head -1)
    if [ -z "$first_global" ]; then
        first_global="0"
    fi

    # Add simple main that calls func_1 and returns a computed value
    cat >> "$output" << EOF

int main(void) {
    func_1();
    return (int)($first_global) & 127;
}
EOF

    rm -f "$csmith_file"
}

# Run a single test and return result
run_test() {
    local issue_type=$1
    local seed=$2
    local test_file="$WORK_DIR/test_$seed.c"
    local ralph_asm="$WORK_DIR/test_$seed.s"
    local macos_asm="$WORK_DIR/test_${seed}_macos.s"
    local obj_file="$WORK_DIR/test_$seed.o"
    local gcc_out="$WORK_DIR/gcc_$seed"
    local ralph_out="$WORK_DIR/ralph_$seed"
    
    # Generate test file
    generate_test "$seed" "$test_file"
    
    # Compile with gcc first
    if ! gcc -O0 -w -o "$gcc_out" "$test_file" 2>/dev/null; then
        echo "SKIP:gcc_fail"
        return
    fi
    
    # Get gcc result
    local gcc_exit=0
    "$gcc_out" >/dev/null 2>&1 || gcc_exit=$?
    
    # Try to compile with ralph-cc
    if ! "$RALPH_CC" -dasm "$test_file" > "$ralph_asm" 2>/dev/null; then
        if [ "$issue_type" = "fail_compile" ]; then
            echo "REGRESS:compile_still_fails"
        else
            echo "REGRESS:compile_fail"
        fi
        return
    fi
    
    # Convert to macOS format
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
    
    # Assemble
    if ! as -o "$obj_file" "$macos_asm" 2>/dev/null; then
        echo "REGRESS:asm_fail"
        return
    fi
    
    # Link
    local SDK_PATH
    SDK_PATH=$(xcrun --show-sdk-path)
    if ! ld -o "$ralph_out" "$obj_file" -lSystem -L"$SDK_PATH/usr/lib" 2>/dev/null; then
        echo "REGRESS:link_fail"
        return
    fi
    
    # Run ralph-cc compiled binary
    local ralph_exit=0
    "$ralph_out" >/dev/null 2>&1 || ralph_exit=$?
    
    # Check for crashes
    if [ "$ralph_exit" -gt 128 ]; then
        if [ "$issue_type" = "crash" ]; then
            echo "REGRESS:still_crashes"
        else
            echo "REGRESS:crash"
        fi
        return
    fi
    
    # Compare exit codes
    if [ "$gcc_exit" -eq "$ralph_exit" ]; then
        echo "PASS"
    else
        if [ "$issue_type" = "mismatch" ]; then
            echo "REGRESS:still_mismatches:gcc=$gcc_exit,ralph=$ralph_exit"
        else
            echo "REGRESS:mismatch:gcc=$gcc_exit,ralph=$ralph_exit"
        fi
    fi
}

# Main
echo "=== Csmith Regression Tests ==="
echo "Testing ${#KNOWN_ISSUES[@]} known issues..."
echo ""

passed=0
regressed=0
skipped=0

for issue in "${KNOWN_ISSUES[@]}"; do
    issue_type="${issue%%:*}"
    seed="${issue##*:}"
    
    result=$(run_test "$issue_type" "$seed")
    
    case "$result" in
        PASS)
            printf "  %-20s seed=%-12s FIXED\n" "$issue_type" "$seed"
            passed=$((passed + 1))
            ;;
        SKIP:*)
            printf "  %-20s seed=%-12s SKIP (%s)\n" "$issue_type" "$seed" "${result#SKIP:}"
            skipped=$((skipped + 1))
            ;;
        REGRESS:*)
            printf "  %-20s seed=%-12s FAIL (%s)\n" "$issue_type" "$seed" "${result#REGRESS:}"
            regressed=$((regressed + 1))
            ;;
    esac
done

echo ""
echo "=== Summary ==="
echo "Passed:    $passed"
echo "Regressed: $regressed"
echo "Skipped:   $skipped"

if [ "$regressed" -gt 0 ]; then
    echo ""
    echo "FAILURE: $regressed regression(s) detected!"
    exit 1
else
    echo ""
    echo "SUCCESS: All known issues stay fixed."
    exit 0
fi
