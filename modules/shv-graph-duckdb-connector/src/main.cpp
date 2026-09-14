#include "connector.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <exception>
#include <iostream>
#include <string>
namespace connector = symphony::shv::duckdb_connector;
namespace engine = symphony::knowledge::engine;
int main(int argc, char **argv) {
  try {
    if (argc == 2 && std::string(argv[1]) == "--descriptor") {
      std::cout << connector::descriptor().dump() << '\n';
      return 0;
    }
    if (argc == 2 && std::string(argv[1]) == "--version") {
      std::cout << connector::engine_id << ' ' << connector::version << '\n';
      return 0;
    }
    if (argc == 2 && std::string(argv[1]) == "--help") {
      std::cout
          << "Usage: symphony-shv-graph-duckdb-connector "
             "[--descriptor|--version|--help]\nOne bounded engine-process.v1 "
             "request; private database root is the current directory.\n";
      return 0;
    }
    if (argc != 1)
      throw engine::Error("argument.unsupported", "unsupported argument", 2);
    auto request = engine::parse_request(
        engine::read_bounded(std::cin, engine::Limits::max_request_bytes),
        connector::engine_id, engine::unix_time_ms());
    try {
      auto result = connector::handle(request);
      std::cout << engine::serialize_response(
          engine::success_response(request, connector::engine_id,
                                   connector::version, std::move(result)));
      return 0;
    } catch (const engine::Error &error) {
      std::cout << engine::serialize_response(engine::error_response(
          request.request_id, request.correlation_id, request.operation,
          connector::engine_id, connector::version, error.code(),
          error.what()));
      return error.exit_status();
    }
  } catch (const engine::Error &error) {
    std::cout << engine::serialize_response(engine::error_response(
        "unavailable", "unavailable", "unavailable", connector::engine_id,
        connector::version, error.code(), error.what()));
    return error.exit_status();
  } catch (const std::exception &) {
    std::cout << engine::serialize_response(engine::error_response(
        "unavailable", "unavailable", "unavailable", connector::engine_id,
        connector::version, "connector.internal",
        "bounded connector processing failed"));
    return 5;
  }
}
