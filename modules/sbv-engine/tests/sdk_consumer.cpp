// Standalone installed C++ consumer: no source checkout or JSON library needed.
#include <chrono>
#include <fstream>
#include <iostream>
#include <iterator>
#include <symphony/sbv/sdk.hpp>
int main(int argc, char **argv) {
  if (argc != 2)
    return 2;
  std::ifstream f(argv[1]);
  if (!f)
    return 2;
  std::string request{std::istreambuf_iterator<char>(f), {}};
  auto response = symphony::sbv::sdk::process(request);
  std::cout << response.json;
  return response.status;
}
