#include <symphony/sbv/interop.hpp>
#include <iostream>
#include <string_view>

int main() {
  namespace interop = symphony::sbv::interop;
  static_assert(__cplusplus >= 202400L, "installed interop requires C++26");
  if (std::string_view(interop::contract_version) != "symphony.sbv.interop.v1")
    return 1;
  interop::BufferDescriptor buffer{"f64", "cpu", "installed-owner", "stream",
                                   16, 0, {2}, {8}};
  const auto admitted = interop::admit_buffer(buffer);
  if (admitted.element_bytes != 8 || admitted.required_bytes != 16 ||
      admitted.empty)
    return 1;
  buffer.storage_bytes = 8;
  try {
    static_cast<void>(interop::admit_buffer(buffer));
  } catch (const std::invalid_argument &) {
    std::cout << "installed interop: exact extent and short-buffer refusal passed\n";
    return 0;
  }
  return 1;
}
