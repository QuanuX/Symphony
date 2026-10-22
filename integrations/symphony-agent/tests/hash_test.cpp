#include "symphony/sha256.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <sys/stat.h>
#include <unistd.h>

namespace fs = std::filesystem;

namespace {
int checks = 0;
void check(bool condition, const char* message) {
    ++checks;
    if (!condition) throw std::runtime_error(message);
}
template <typename Action> void rejects(Action action, const char* message) {
    bool rejected = false;
    try { action(); } catch (const std::exception&) { rejected = true; }
    check(rejected, message);
}
struct TemporaryDirectory {
    std::string path;
    ~TemporaryDirectory() {
        std::error_code ignored;
        fs::remove_all(path, ignored);
    }
};
} // namespace

int main() {
    char pattern[] = "/tmp/symphony-hash-XXXXXX";
    const char* created = ::mkdtemp(pattern);
    if (!created) { std::cerr << "hash fixture directory creation failed\n"; return 1; }
    TemporaryDirectory temporary{created};
    const auto input = temporary.path + "/input";
    const auto fifo = temporary.path + "/fifo";
    try {
        {
            std::ofstream stream(input);
            stream << "abc";
            if (!stream) throw std::runtime_error("hash fixture creation failed");
        }
        check(symphony::file_sha256(input) ==
                  "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
              "SHA-256 abc known vector must match");
        rejects([&] { symphony::file_sha256(input, 2); }, "hash byte limit must reject oversized input");
        if (::mkfifo(fifo.c_str(), 0600) != 0) throw std::runtime_error("FIFO fixture creation failed");
        rejects([&] { symphony::file_sha256(fifo); }, "FIFO must reject without waiting for a writer");
        {
            std::ofstream stream(input, std::ios::trunc);
            if (!stream) throw std::runtime_error("empty fixture creation failed");
        }
        check(symphony::file_sha256(input) ==
                  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "SHA-256 empty known vector must match");
        rejects([&] { symphony::file_sha256(temporary.path); }, "directory must reject as digest input");
        std::cout << checks << " hash checks passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "hash test failure: " << error.what() << '\n';
        return 1;
    }
}
