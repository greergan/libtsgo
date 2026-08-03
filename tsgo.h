#pragma once
#ifdef __cplusplus
#include <string_view>
#include <cstdlib>

struct GoStr {
    char* data = nullptr;
    GoStr() noexcept : data(nullptr) {}
    GoStr(char* ptr) : data(ptr) {}
    GoStr(const GoStr&) = delete;
    GoStr& operator=(const GoStr&) = delete;
    GoStr(GoStr&& other) noexcept : data(other.data) { other.data = nullptr; }
    GoStr& operator=(GoStr&& other) noexcept { if(this != &other) { free(data); data = other.data; other.data = nullptr; } return *this; }
    ~GoStr() { free(data); }
    std::string_view view() const { return data ? data : ""; }
};

#else
#include <stdlib.h>

typedef struct {
    char* data;
} GoStr;

static inline void GoStr_free(GoStr s) { free(s.data); }

#endif
