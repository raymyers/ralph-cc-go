/* recursive.c - Recursive factorial
 *
 * This tests:
 * - Recursive function calls
 * - Stack growth with multiple frames
 * - Callee-saved register restoration
 * - Base case termination
 *
 * Expected: factorial(5) = 120
 * Return value is clamped to 255 by exit code
 */

int factorial(int n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

int main() {
    return factorial(5);  /* Returns 120 */
}
