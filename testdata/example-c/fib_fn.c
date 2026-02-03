#include <stdio.h>

void header(const int n) {
     printf("First %d Fibonacci numbers:\n", n);
}

int main() {
    int n = 30;
    long long first = 0, second = 1, next;

    header(n);

    for (int i = 1; i <= n; i++) {
        printf("%lld ", first);
        next = first + second;
        first = second;
        second = next;
    }

    return 0;
}
