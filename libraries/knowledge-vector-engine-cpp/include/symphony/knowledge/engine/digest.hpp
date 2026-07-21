#pragma once

#include <span>
#include <string>
#include <string_view>

namespace symphony::knowledge::engine {

[[nodiscard]] std::string sha256_hex(std::span<const unsigned char> bytes);
[[nodiscard]] std::string sha256_hex(std::string_view text);
[[nodiscard]] std::string tagged_sha256(std::string_view text);

}
