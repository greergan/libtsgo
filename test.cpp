#include <print>
#include <string>
#include <vector>
#include "tsgo.h"

struct TestCase {
    int id;
    std::string uri;
};

const std::string expected = "import console from 'console';\nconsole.log(\"Hello, World!\");\n";

const std::vector<TestCase> cases = {
    {1, "http://forgejo:8000/src/hello_world.ts"},
    {2, "file://../typescript_samples/src/hello_world.ts"},
    {3, "http://forgejo/greergan/typescript_samples/raw/branch/master/src/hello_world.ts"},
    {4, "https://raw.githubusercontent.com/greergan/typescript_samples/master/src/hello_world.ts"},
    {5, "https://codeberg.org/greergan/typescript_samples/raw/branch/master/src/hello_world.ts"},
};

int main(int argc, char** argv) {
    for (const auto& tc : cases) {
        std::println("fetching {}", tc.uri);
        GoStr result = fetch_and_transpile(const_cast<char*>(tc.uri.c_str()));
        if (result.view() != expected) {
            std::println("Test {} Failed", tc.id);
            std::println("Expected:\n{}", expected);
            std::println("Received:\n{}", result.view());
            return tc.id;
        }
        std::println("processed {}", tc.uri);
    }

    return 0;
}
