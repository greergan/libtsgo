# libtsgo

A C and C++ callable static library that wraps the [microsoft/typescript-go](https://github.com/microsoft/typescript-go) compiler, enabling high-performance TypeScript compilation from C or C++.

## Table of Contents

- [Typescript Definition Files](#typescript-definition-files)
- [Compiler Options](#compiler-options)
- [API](#api)
  - [GoStr helper](#gostr-helper)
  - [transpile](#transpile)
  - [build](#build)
- [Requirements](#requirements)
- [Build](#build-1)
- [Examples](#examples)
  - [transpile](#transpile-1)
  - [build](#build-2)

## Typescript Definition Files

| Directory | Purpose | When |
|---|---|---|
| `typescript-go` | All `microsoft/typescript-go` compiler libraries | Embedded at compile time into the static archive |
| `lib/` | TypeScript standard library `.d.ts` files | Embedded at compile time into the static archive |
| `types/` | User-provided type definitions | Loaded at runtime from the working directory |

[↑ Top](#table-of-contents)

## Compiler Options

The following `compilerOptions` are embedded at compile time:

| Option | Value |
|---|---|
| `target` | `ESNext` |
| `module` | `ESNext` |
| `moduleResolution` | `bundler` |
| `allowJs` | `true` |
| `checkJs` | `true` |
| `removeComments` | `false` |
| `forceConsistentCasingInFileNames` | `true` |
| `strict` | `true` |
| `noUnusedLocals` | `true` |
| `noUnusedParameters` | `true` |
| `noFallthroughCasesInSwitch` | `true` |
| `noImplicitOverride` | `true` |
| `skipDefaultLibCheck` | `true` |

[↑ Top](#table-of-contents)

## API

### GoStr helper

A lightweight wrapper for strings returned by the library. Include `tsgo.h` to use it.

```c
#include "tsgo.h"
```

In C++, `GoStr` is an RAII struct — the destructor calls `free()` automatically.

```cpp
GoStr result(transpile(...));
std::cout << result.view() << std::endl;
// freed on scope exit
```

In C, `GoStr` is a plain struct — call `GoStr_free()` manually.

```c
GoStr result;
result.p = transpile(...);
printf("%s\n", result.p ? result.p : "");
GoStr_free(result);
```

### `transpile`

Compiles a single TypeScript file in-memory.

```c
char* transpile(
    char* fileName,   // virtual file name e.g. "input.ts"
    char* tsCode,     // TypeScript source
    char* dtsCode,    // optional .d.ts declarations, or NULL
    char* outDir      // output directory, or NULL for in-memory result
);
```

Returns emitted JavaScript.  
Caller must `free()` the returned string, or use the provided `GoStr` helper above.

### `build`

Compiles all `.ts` files in a source tree. Diagnostics and errors are printed to stdout.

```c
void build(
    char* srcDir,   // source directory e.g. "src"
    char* outDir    // output directory e.g. "dist"
);
```

[↑ Top](#table-of-contents)

## Requirements

- Go 1.26+
- gcc
- g++
- git
- make

[↑ Top](#table-of-contents)

## Build

```bash
git clone https://github.com/greergan/tsgo_cpp.git
cd tsgo_cpp
make
```

This will:
- Clone `microsoft/typescript-go` at branch `typescript/v7.0.2`
- Build `libtsgo.a` and `libtsgo.h`

[↑ Top](#table-of-contents)

## Examples

### transpile

#### C — in-memory result

```c
#include "tsgo.h"

const char* ts = "const x = 42;\nconsole.log(x);\n";
GoStr result;
result.p = transpile((char*)"input.ts", (char*)ts, NULL, NULL);
printf("%s\n", result.p ? result.p : "");
GoStr_free(result);
```

#### C++ — in-memory result

```cpp
#include "tsgo.h"

std::string ts = "const x: number = 42;\nconsole.log(x);\n";
GoStr result(transpile(
    const_cast<char*>("input.ts"),
    const_cast<char*>(ts.c_str()),
    nullptr,
    nullptr
));
std::cout << result.view() << std::endl;
```

#### C — emit to disk

```c
#include "tsgo.h"

const char* ts = "const x = 42;\nconsole.log(x);\n";
GoStr result;
result.p = transpile((char*)"input.ts", (char*)ts, NULL, (char*)"dist");
GoStr_free(result);
```

#### C++ — emit to disk

```cpp
#include "tsgo.h"

std::string ts = "const x: number = 42;\nconsole.log(x);\n";
transpile(
    const_cast<char*>("input.ts"),
    const_cast<char*>(ts.c_str()),
    nullptr,
    const_cast<char*>("dist")
);
```

#### C — with .d.ts declarations

```c
#include "tsgo.h"

const char* dts = "declare function add(a: number, b: number): number;\n";
const char* ts  = "const result = add(1, 2);\nconsole.log(result);\n";
GoStr result;
result.p = transpile((char*)"input.ts", (char*)ts, (char*)dts, NULL);
printf("%s\n", result.p ? result.p : "");
GoStr_free(result);
```

#### C++ — with .d.ts declarations

```cpp
#include "tsgo.h"

std::string dts = "declare function add(a: number, b: number): number;\n";
std::string ts  = "const result = add(1, 2);\nconsole.log(result);\n";
GoStr result(transpile(
    const_cast<char*>("input.ts"),
    const_cast<char*>(ts.c_str()),
    const_cast<char*>(dts.c_str()),
    nullptr
));
std::cout << result.view() << std::endl;
```

### build

#### C

```c
#include "tsgo.h"

build((char*)"src", (char*)"dist");
```

#### C++

```cpp
#include "tsgo.h"

build(
    const_cast<char*>("src"),
    const_cast<char*>("dist")
);
```

[↑ Top](#table-of-contents)
