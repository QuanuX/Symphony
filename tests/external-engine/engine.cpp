#include "calculation.hpp"
#include <iostream>
#include <symphony/knowledge/engine/protocol.hpp>
namespace e = symphony::knowledge::engine;
int main(int argc, char **argv) {
  if (argc == 2 && std::string(argv[1]) == "--version") {
    std::cout << beta_example::id << " " << BETA_EXAMPLE_VERSION << '\n';
    return 0;
  }
  try {
    if (argc != 1)
      throw std::runtime_error("one request on stdin required");
    const auto request =
        e::parse_request(e::read_bounded(std::cin, 32768), beta_example::id,
                         e::unix_time_ms(), 4096, true);
    const auto &p = request.payload;
    if (request.operation != "empirical_distribution" || !p.is_object() ||
        p.size() != 2 ||
        p.at("protocol") != "quanux.beta.empirical-price-input.v1" ||
        !p.at("prices_nanos").is_array())
      throw std::runtime_error("unsupported input or operation");
    std::vector<std::int64_t> prices;
    for (const auto &v : p.at("prices_nanos"))
      prices.push_back(beta_example::integer(v));
    const auto result = beta_example::calculate(prices);
    std::cout << e::serialize_response(e::success_response(
                     request, beta_example::id, BETA_EXAMPLE_VERSION, result))
              << '\n';
    return 0;
  } catch (const std::exception &) {
    std::cerr << "empirical example refused input\n";
    return 2;
  }
}
