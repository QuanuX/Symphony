#include "source.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <exception>
#include <iostream>
#include <string>
#include <utility>

namespace shv = symphony::knowledge::shv_source;
namespace engine = symphony::knowledge::engine;

namespace {

int emit_error(const engine::Error &error) {
  try {
    std::cout << engine::serialize_response(engine::error_response(
        "unavailable", "unavailable", "unavailable", shv::engine_id,
        shv::version, error.code(), error.what()));
  } catch (const std::exception &) {
    return 5;
  }
  return error.exit_status();
}

} // namespace

int main(int argc, char **argv) {
  if (argc == 2) {
    const std::string argument = argv[1];
    if (argument == "--help") {
      std::cout
          << "Usage: symphony-shv-source [--help|--version|--descriptor]\n"
             "Without arguments, reads one bounded process request from "
             "standard input.\n";
      return 0;
    }
    if (argument == "--version") {
      std::cout << shv::engine_id << ' ' << shv::version << '\n';
      return 0;
    }
    if (argument == "--descriptor") {
      std::cout << shv::descriptor().dump() << '\n';
      return 0;
    }
    return emit_error(
        engine::Error("argument.unsupported", "unsupported argument", 2));
  }
  if (argc != 1) {
    return emit_error(
        engine::Error("argument.count", "unexpected argument count", 2));
  }

  try {
    const auto input =
        engine::read_bounded(std::cin, engine::Limits::max_request_bytes);
    const auto request =
        engine::parse_request(input, shv::engine_id, engine::unix_time_ms());
    try {
      auto result = shv::handle_request(request);
      std::cout << engine::serialize_response(engine::success_response(
          request, shv::engine_id, shv::version, std::move(result)));
      return 0;
    } catch (const engine::Error &error) {
      std::cout << engine::serialize_response(engine::error_response(
          request.request_id, request.correlation_id, request.operation,
          shv::engine_id, shv::version, error.code(), error.what()));
      return error.exit_status();
    }
  } catch (const engine::Error &error) {
    return emit_error(error);
  } catch (const std::exception &) {
    return emit_error(engine::Error("internal.failure",
                                    "bounded request processing failed", 5));
  }
}
