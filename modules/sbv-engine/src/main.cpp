#include <iostream>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sbv.hpp>
namespace e = symphony::knowledge::engine;
int main(int argc, char **argv) {
  const std::string id = "symphony-sbv", version = symphony::sbv::version;
  try {
    if (argc == 2 && std::string(argv[1]) == "--descriptor") {
      std::cout << symphony::sbv::descriptor().dump() << '\n';
      return 0;
    }
    if (argc == 2 && std::string(argv[1]) == "--version") {
      std::cout << id << ' ' << version << '\n';
      return 0;
    }
    if (argc == 2 && std::string(argv[1]) == "--help") {
      std::cout << "symphony-sbv: one engine-process.v1 JSON request on stdin; "
                   "--descriptor, --version\n";
      return 0;
    }
    if (argc != 1)
      throw e::Error("invocation.arguments", "unsupported invocation", 2);
    const auto request = e::parse_request(
        e::read_bounded(std::cin, e::Limits::max_request_bytes), id,
        e::unix_time_ms());
    try {
      auto result =
          request.operation == "descriptor"
              ? symphony::sbv::descriptor()
              : symphony::sbv::dispatch(request.operation, request.payload,
                                        request.deadline_unix_ms);
      if (request.operation == "descriptor" &&
          (!request.payload.is_object() || !request.payload.empty()))
        throw e::Error("descriptor.input", "empty payload required", 2);
      // A run may have committed its artifact just before its deadline. The
      // result includes its digest/path; never mislabel a published artifact as
      // absent.
      std::cout << e::serialize_response(
                       e::success_response(request, id, version, result))
                << '\n';
      return 0;
    } catch (const e::Error &x) {
      std::cout << e::serialize_response(e::error_response(
                       request.request_id, request.correlation_id,
                       request.operation, id, version, x.code(), x.what()))
                << '\n';
      return x.exit_status();
    } catch (...) {
      std::cout << e::serialize_response(e::error_response(
                       request.request_id, request.correlation_id,
                       request.operation, id, version, "sbv.failure",
                       "SBV operation failed; inspect output destination "
                       "before retrying a write"))
                << '\n';
      return 4;
    }
  } catch (const e::Error &x) {
    std::cout << e::serialize_response(e::error_response(
                     "unavailable", "unavailable", "unavailable", id, version,
                     x.code(), "SBV request rejected"))
              << '\n';
    return x.exit_status();
  } catch (...) {
    std::cerr << "SBV request failed\n";
    return 4;
  }
}
