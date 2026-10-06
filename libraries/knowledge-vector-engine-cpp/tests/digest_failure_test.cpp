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
    require(symphony::knowledge::engine::sha256_hex(std::string(55, 'a')) == "9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318", "padding/large reference 55");
    require(symphony::knowledge::engine::sha256_hex(std::string(56, 'a')) == "b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a", "padding/large reference 56");
    require(symphony::knowledge::engine::sha256_hex(std::string(63, 'a')) == "7d3e74a05d7db15bce4ad9ec0658ea98e3f06eeecf16b4c6fff2da457ddc2f34", "padding/large reference 63");
    require(symphony::knowledge::engine::sha256_hex(std::string(64, 'a')) == "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb", "padding/large reference 64");
    require(symphony::knowledge::engine::sha256_hex(std::string(65, 'a')) == "635361c48bb9eab14198e76ea8ab7f1a41685d6ad62aa9146d301d4f17eb0ae0", "padding/large reference 65");
    require(symphony::knowledge::engine::sha256_hex(std::string(119, 'a')) == "31eba51c313a5c08226adf18d4a359cfdfd8d2e816b13f4af952f7ea6584dcfb", "padding/large reference 119");
    require(symphony::knowledge::engine::sha256_hex(std::string(120, 'a')) == "2f3d335432c70b580af0e8e1b3674a7c020d683aa5f73aaaedfdc55af904c21c", "padding/large reference 120");
    require(symphony::knowledge::engine::sha256_hex(std::string(127, 'a')) == "c57e9278af78fa3cab38667bef4ce29d783787a2f731d4e12200270f0c32320a", "padding/large reference 127");
    require(symphony::knowledge::engine::sha256_hex(std::string(128, 'a')) == "6836cf13bac400e9105071cd6af47084dfacad4e5e302c94bfed24e013afb73e", "padding/large reference 128");
    require(symphony::knowledge::engine::sha256_hex(std::string(129, 'a')) == "c12cb024a2e5551cca0e08fce8f1c5e314555cc3fef6329ee994a3db752166ae", "padding/large reference 129");
    require(symphony::knowledge::engine::sha256_hex(std::string(1000000, 'a')) == "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0", "padding/large reference 1000000");
    {
        const std::string input(1000000, 'a');
        allocation_calls = 0;
        fail_at = 100;
        const auto digest = symphony::knowledge::engine::sha256_hex(input);
        fail_at = -1;
        require(allocation_calls == 1, "hash allocates only the output, independent of input size");
        require(digest.size() == 64, "complete large digest");
    }
    allocation_failure_preserves_digest_contract();
    require(live_allocations.load() == before, "all test allocations released");
}
