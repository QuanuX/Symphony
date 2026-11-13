// Deliberately synthetic trade prices; only the DBN wire header/record layout
// comes from the attributed public seed. No live, licensed or private data.
#include "fixtures/public_fixture.hpp"
#include <fstream>
#include <stdexcept>
int main(int argc, char **argv) {
  if (argc != 2)
    return 2;
  const auto seed = public_fixture::fixture();
  std::vector<uint8_t> bytes(seed.begin(), seed.begin() + 360);
  unsigned ordinal = 0;
  auto event = [&](char action, char side, uint64_t order, int64_t price,
                   uint32_t size) {
    auto offset = bytes.size();
    bytes.insert(bytes.end(), seed.begin() + 360, seed.begin() + 416);
    auto put = [&](unsigned field, uint64_t value, unsigned length) {
      for (unsigned i = 0; i < length; i++)
        bytes[offset + field + i] = static_cast<uint8_t>(value >> (8 * i));
    };
    put(8, 1609160400000000000ULL + ordinal * 100, 8);
    put(40, 1609160400000000000ULL + ordinal * 100, 8);
    put(16, order, 8);
    put(24, static_cast<uint64_t>(price), 8);
    put(32, size, 4);
    put(52, ordinal, 4);
    bytes[offset + 36] = 128;
    bytes[offset + 37] = 0;
    bytes[offset + 38] = action;
    bytes[offset + 39] = side;
    ++ordinal;
  };
  event('R', 'N', 0, 0, 0);
  event('A', 'B', 1, 99000000000LL, 20);
  event('A', 'A', 2, 101000000000LL, 20);
  for (const auto price :
       {100, 95, 105, 96, 104, 97, 103, 98, 102, 99, 101, 100})
    event('T', 'B', 0, price * 1000000000LL, 1);
  std::ofstream output(argv[1], std::ios::binary);
  output.write(reinterpret_cast<const char *>(bytes.data()),
               static_cast<std::streamsize>(bytes.size()));
  return output.good() ? 0 : 2;
}
