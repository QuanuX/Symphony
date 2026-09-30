#include <symphony/snv/common.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <iostream>
#include <sstream>
#include <stdexcept>

namespace {
namespace engine = symphony::knowledge::engine;
using symphony::snv::Json;

struct Streams {
  std::istringstream input;
  std::ostringstream output;
  std::streambuf* old_input;
  std::streambuf* old_output;
  explicit Streams(const Json& request)
      : input(request.dump()), old_input(std::cin.rdbuf(input.rdbuf())),
        old_output(std::cout.rdbuf(output.rdbuf())) {}
  ~Streams() { std::cin.rdbuf(old_input); std::cout.rdbuf(old_output); }
};
void require(bool value, const char* message) {
  if (!value) throw std::runtime_error(message);
}
Json request() {
  return Json{{"protocol", engine::process_protocol_v1}, {"request_id", "known-request"},
    {"correlation_id", "known-correlation"}, {"operation", "inspect"},
    {"target_engine", "symphony-sciv"},
    {"deadline_unix_ms", engine::unix_time_ms() + 5000}, {"payload", Json::object()}};
}
void unexpected_exception(const symphony::snv::Handler& handler) {
  Streams streams(request());
  char program[] = "mechanics-test";
  char* arguments[] = {program};
  const int status = symphony::snv::run(1, arguments, "sciv", "0.1.0-dev", {}, handler);
  const auto response = engine::parse_bounded_json(streams.output.str(), engine::Limits::max_response_bytes);
  require(status == 1, "unexpected exception must be a nonzero process result");
  require(response.at("request_id") == "known-request", "request identity lost");
  require(response.at("correlation_id") == "known-correlation", "correlation identity lost");
  require(response.at("operation") == "inspect", "operation identity lost");
  require(response.at("outcome") == "error" && response.at("result").is_null(), "invalid failure shape");
  require(response.at("error").at("code") == "internal.failure", "unexpected error classification");
  require(response.at("error").at("message") == "bounded operation failed", "exception text escaped");
  require(streams.output.str().find("private-evidence") == std::string::npos, "private exception marker leaked");
}
}
int main() {
  try {
    unexpected_exception([](std::string_view, const Json&, std::int64_t) -> Json {
      throw std::runtime_error("private-evidence");
    });
    unexpected_exception([](std::string_view, const Json&, std::int64_t) -> Json {
      throw 17;
    });
    std::cout << "common mechanics: correlated unexpected failures PASS\n";
    return 0;
  } catch (const std::exception& failure) {
    std::cerr << failure.what() << '\n';
    return 1;
  }
}
