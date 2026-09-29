#include <cstdlib>
#include <string>
#include <symphony/source/json.hpp>
#include <symphony/source/support.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  nlohmann::json out = {{"preserved", true}};
  std::string bad = "{\"x\":1,\"x\":2}";
  auto status = source::parse_json(
      {reinterpret_cast<const std::uint8_t *>(bad.data()), bad.size()},
      {1024, 128, 128, 8}, out);
  check(status == source::Status::malformed && out.contains("preserved"));
  check(source::decimal("0.000000000000000001") && !source::date("2026-02-29"));
}
int main() { installed_boundary(); }
