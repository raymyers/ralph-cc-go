/* many_args.c - Function with >8 arguments
 *
 * This tests:
 * - ARM64 calling convention: X0-X7 for args 1-8
 * - Stack passing for arg 9+
 * - Correct stack offset calculation
 *
 * sum9(1,2,3,4,5,6,7,8,9) = 45
 * Expected: 45
 */

int sum9(int a, int b, int c, int d, int e, int f, int g, int h, int i) {
    return a + b + c + d + e + f + g + h + i;
}

int main() {
    return sum9(1, 2, 3, 4, 5, 6, 7, 8, 9);  /* Returns 45 */
}
