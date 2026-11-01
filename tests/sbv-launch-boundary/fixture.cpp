// Mechanical census fixture, matching the owner's census regression. The wire
// seed is the attributed public Databento test fixture copied beside this file.
#include "public_fixture.hpp"
#include <fstream>
#include <stdexcept>
int main(int argc, char **argv) {
  if (argc != 2) return 2;
  const auto seed = public_fixture::fixture();
  std::vector<std::uint8_t> bytes(seed.begin(), seed.begin() + 360);
  unsigned count = 0;
  auto event = [&](char action, std::uint64_t price, unsigned size, char side,
                   std::uint64_t id) {
    const auto offset = bytes.size();
    bytes.insert(bytes.end(), seed.begin() + 360, seed.begin() + 416);
    auto put = [&](unsigned field, std::uint64_t value, unsigned length) {
      for (unsigned i = 0; i < length; ++i)
        bytes[offset + field + i] = static_cast<std::uint8_t>(value >> (i * 8));
    };
    put(8, 1609160400000000000ULL + count * 100, 8);
    put(40, 1609160400000000000ULL + count * 100, 8);
    put(16, id, 8); put(24, price, 8); put(32, size, 4); put(52, count, 4);
    bytes[offset + 36] = 128; bytes[offset + 37] = 0;
    bytes[offset + 38] = action; bytes[offset + 39] = side;
    ++count;
  };
  event('R', 0, 0, 'N', 0);
  event('A', 99000000000ULL, 20, 'B', 1);
  event('A', 101000000000ULL, 20, 'A', 2);
  for (auto price : {100, 95, 105, 96, 104, 97, 103, 98, 102, 99, 101, 100})
    event('T', price * 1000000000ULL, 1, 'B', 0);
  std::ofstream out(argv[1], std::ios::binary);
  out.write(reinterpret_cast<const char *>(bytes.data()), bytes.size());
  if (!out.good() || bytes.size() != 1200) throw std::runtime_error("fixture write");
}
