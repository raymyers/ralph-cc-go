/* negative.c - Negative number handling
 *
 * This tests:
 * - Negative integer constants
 * - Subtraction resulting in negative
 * - Comparison with negative numbers
 * - Sign preservation in arithmetic
 *
 * Expected: 42 (abs of -42 computed manually)
 */

int main() {
    int x = -42;
    int y = 0 - x;  /* y = 42 */
    if (x < 0) {
        return y;
    }
    return 0;
}
