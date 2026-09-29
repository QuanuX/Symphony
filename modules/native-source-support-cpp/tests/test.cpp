#include <cstdio>
#include <cstdlib>
#include <symphony/source/json.hpp>
using namespace symphony::source;
void check(bool p) {
  if (!p)
    std::abort();
}
Bytes bytes(std::string_view s) {
  return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}
int main() {
  JsonLimits limits{65536, 1024, 4096, 16};
  nlohmann::json out = {{"sentinel", 1}};
  check(parse_json(
            bytes("{\"v\":1.234567890123456789,\"u\":18446744073709551615}"),
            limits, out) == Status::ok);
  check(out.at("u").get<std::uint64_t>() == UINT64_MAX);
  auto previous = out;
  check(parse_json(bytes("{\"x\":1,\"x\":2}"), limits, out) ==
            Status::malformed &&
        out == previous);
  check(parse_json(bytes("[1,2] trailing"), limits, out) == Status::malformed &&
        out == previous);
  auto small = limits;
  small.values = 2;
  check(parse_json(bytes("[1,2]"), small, out) == Status::limit &&
        out == previous);
  small = limits;
  small.depth = 1;
  check(parse_json(bytes("[[[0]]]"), small, out) == Status::limit);
  check(date("2024-02-29") && !date("2023-02-29") && !date("0000-01-01"));
  check(decimal("-0.0000000000000000001") && !decimal("NaN") && !decimal(".") &&
        !decimal("1e"));
  check(encode("A&B/+") == "A%26B%2F%2B");
  HttpRequest request{"http://localhost/path", "", false,
                      HttpRequest::Credential::none};
  HttpResponse response;
  check(fetch(request, nullptr, {100, 50, 1024, 1024}, {}, response) ==
        Status::invalid_argument);
  request.endpoint = "https://localhost/path";
  std::stop_source stop;
  stop.request_stop();
  check(fetch(request, nullptr, {100, 50, 1024, 1024}, stop.get_token(),
              response) == Status::cancelled);
  std::puts("source support: JSON bounds/duplicates, dates/decimals, HTTP "
            "validation passed");
}
