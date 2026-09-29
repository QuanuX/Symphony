#include <symphony/knowledge/engine/digest.hpp>

#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <new>
#include <string>

namespace {
std::atomic<int> fail_at{-1};
std::atomic<int> allocation_calls{0};
std::atomic<std::size_t> live_allocations{0};
}

void* operator new(std::size_t size) {
    const auto selected = fail_at.load();
    if (selected >= 0 && allocation_calls.fetch_add(1) == selected)
        throw std::bad_alloc();
    if (void* result = std::malloc(size == 0 ? 1 : size)) {
        ++live_allocations;
        return result;
    }
    throw std::bad_alloc();
}
void* operator new[](std::size_t size) { return ::operator new(size); }
void operator delete(void* pointer) noexcept {
    if (pointer) --live_allocations;
    std::free(pointer);
}
void operator delete[](void* pointer) noexcept { ::operator delete(pointer); }

namespace {
void require(bool value, const char* message) {
    if (value) return;
    std::fprintf(stderr, "foundation digest failure test: %s\n", message);
    std::abort();
}

void allocation_failure_preserves_digest_contract() {
    const std::string input(80, 'x');
    const std::string expected = "d929cdeea7e6a0f46b59448d36a6fd491df8647cf292040508162ee5f6435ab8";
    const std::string tagged = "sha256:" + expected;
    unsigned rejections = 0;
    for (unsigned operation = 0; operation < 3; ++operation) {
        bool complete = false;
        for (int selected = 0; selected < 64; ++selected) {
            std::string output = "unchanged";
            const auto before = live_allocations.load();
            bool rejected = false;
            allocation_calls = 0;
            fail_at = selected;
            try {
                if (operation == 0) {
                    output = symphony::knowledge::engine::sha256_hex(std::string_view(input));
                } else if (operation == 1) {
                    output = symphony::knowledge::engine::sha256_hex(std::span<const unsigned char>(
                        reinterpret_cast<const unsigned char*>(input.data()), input.size()));
                } else {
                    output = symphony::knowledge::engine::tagged_sha256(input);
                }
            } catch (const std::bad_alloc&) {
                rejected = true;
            } catch (...) {
                fail_at = -1;
                require(false, "unexpected exception type");
            }
            fail_at = -1;
            if (rejected) {
                ++rejections;
                require(output == "unchanged", "failure publishes no partial digest");
                require(live_allocations.load() == before, "failed hash releases allocations");
                continue;
            }
            require(output == (operation == 2 ? tagged : expected), "successful result is the complete exact digest");
            complete = true;
            break;
        }
        require(complete, "all allocation points exhausted");
    }
    require(rejections > 0, "allocation failures exercised");
    std::printf("foundation digest: %u rejected allocation points\n", rejections);
}
}

int main() {
    const auto before = live_allocations.load();
    require(symphony::knowledge::engine::sha256_hex(std::string_view{}) ==
        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "empty golden hash");
    require(symphony::knowledge::engine::sha256_hex(std::string_view("abc")) ==
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", "abc golden hash");
    allocation_failure_preserves_digest_contract();
    require(live_allocations.load() == before, "all test allocations released");
}
