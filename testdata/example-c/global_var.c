/* global_var.c - Global variable access
 *
 * This tests:
 * - Global variable in data section
 * - Reading from global
 * - Writing to global  
 * - PC-relative addressing (adrp/ldr)
 *
 * Expected: 42 + 8 = 50
 */

int g = 42;

int main() {
    int x = g;
    g = g + 8;
    return g;  /* Returns 50 */
}
