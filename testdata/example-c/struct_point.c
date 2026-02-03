/* struct_point.c - Struct member access
 *
 * This tests:
 * - Struct field layout
 * - Member access (dot operator)
 * - Struct as local variable
 *
 * Expected: 10 + 32 = 42
 */

struct Point {
    int x;
    int y;
};

int main() {
    struct Point p;
    p.x = 10;
    p.y = 32;
    return p.x + p.y;  /* Returns 42 */
}
