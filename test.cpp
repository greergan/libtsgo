#include <print>
#include <string>
#include <vector>
#include "tsgo.h"
#include "libtsgo.h"

struct TestCase {
    int id;
    std::string uri;
};

const std::vector<TestCase> cases = {
    //{1, "file://examples/src/hello_world.ts"},
    {2, "https://raw.githubusercontent.com/greergan/typescript_samples/master/src/hello_world.ts"},
    {3, "https://codeberg.org/greergan/typescript_samples/raw/branch/master/src/hello_world.ts"},
};

int main(int argc, char** argv) {
    for (const auto& tc : cases) {
        std::println("running test for => {}", tc.uri);
        GoStr result = fetch_and_transpile(const_cast<char*>(tc.uri.c_str()));
        std::println("results for => {}\n{}", tc.uri, result.view());
    }
    return 0;
}
