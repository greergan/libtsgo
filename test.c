#include <stdio.h>
#include <string.h>
#include "tsgo.h"
#include "libtsgo.h"

typedef struct {
    int id;
    const char* uri;
} TestCase;

static const TestCase cases[] = {
    //{1, "file://examples/src/hello_world.ts"},
    {2, "https://raw.githubusercontent.com/greergan/typescript_samples/master/src/hello_world.ts"},
    {3, "https://codeberg.org/greergan/typescript_samples/raw/branch/master/src/hello_world.ts"},
};

int main(int argc, char** argv) {
    int n = sizeof(cases) / sizeof(cases[0]);
    for (int i = 0; i < n; i++) {
        printf("running test for => %s\n", cases[i].uri);
        GoStr result;
        result.p = fetch_and_transpile((char*)cases[i].uri);
        printf("results for => %s\n%s\n", cases[i].uri, result.p ? result.p : "");
        GoStr_free(result);
    }
    return 0;
}
