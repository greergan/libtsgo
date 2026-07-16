#include <print>
#include <string>
#include "tsgo.h"

int main(int argc, char** argv) {
    std::string uri = "file://../typescript_samples/src/hello_world.ts";
    GoStr result1 = fetch_and_transpile(const_cast<char*>(uri.c_str()));
    std::println("{}", result1.view());

    uri = "http://forgejo/greergan/typescript_samples/raw/branch/master/src/hello_world.ts";
    GoStr result2 = fetch_and_transpile(const_cast<char*>(uri.c_str()));
    std::println("{}", result2.view());
}
