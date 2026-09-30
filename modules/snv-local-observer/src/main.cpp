#include <iostream>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/snv/local_observer.hpp>
#ifdef OBSERVER_GENERATED_INTERFACE
#include "interface.generated.hpp"
#endif
namespace observer = symphony::snv::local_observer;
namespace engine = symphony::knowledge::engine;
int main(int argc, char **argv) {
  try {
#ifdef OBSERVER_GENERATED_INTERFACE
    const auto &actual = observer::operations();
    const auto expected = observer::interface::interface_operations();
    if (engine::administration_operation_descriptors(actual) !=
        engine::administration_operation_descriptors(expected))
      return 2;
#endif
    if (argc == 2) {
      const std::string_view flag = argv[1];
      if (flag == "--descriptor") {
        std::cout << observer::descriptor().dump() << '\n';
        return 0;
      }
      if (flag == "--version") {
        std::cout << observer::engine_id << ' ' << observer::version << '\n';
        return 0;
      }
      if (flag == "--help") {
        std::cout << observer::engine_id
                  << ": one explicitly requested bounded Linux proc/sysfs "
                     "observation; --descriptor, --version\n";
        return 0;
      }
    }
    if (argc != 1)
      throw engine::Error("invocation.arguments", "unsupported invocation", 2);
    const auto bytes =
        engine::read_bounded(std::cin, engine::Limits::max_request_bytes);
    const auto request = engine::parse_request(bytes, observer::engine_id,
                                               engine::unix_time_ms(),
                                               observer::max_json_values);
    try {
      observer::Json result;
      if (request.operation == "descriptor") {
        if (!request.payload.is_object() || !request.payload.empty())
          throw engine::Error("descriptor.input",
                              "descriptor requires empty payload", 2);
        result = observer::descriptor();
      } else
        result = observer::handle(request.operation, request.payload,
                                  request.deadline_unix_ms);
      if (engine::unix_time_ms() >= request.deadline_unix_ms)
        throw engine::Error("deadline_exceeded", "observer deadline exceeded",
                            3);
      std::cout << engine::serialize_response(
                       engine::success_response(request, observer::engine_id,
                                                observer::version, result),
                       observer::max_json_values)
                << '\n';
      return 0;
    } catch (const engine::Error &e) {
      std::cout << engine::serialize_response(
                       engine::error_response(
                           request.request_id, request.correlation_id,
                           request.operation, observer::engine_id,
                           observer::version, e.code(), e.what()),
                       observer::max_json_values)
                << '\n';
      return e.exit_status();
    } catch (const std::exception &) {
      std::cout << engine::serialize_response(
                       engine::error_response(
                           request.request_id, request.correlation_id,
                           request.operation, observer::engine_id,
                           observer::version, "internal.failure",
                           "bounded observer failed"),
                       observer::max_json_values)
                << '\n';
      return 1;
    }
  } catch (const engine::Error &e) {
    std::cout << engine::serialize_response(engine::error_response(
                     "unavailable", "unavailable", "unavailable",
                     observer::engine_id, observer::version, e.code(),
                     e.what()))
              << '\n';
    return e.exit_status();
  } catch (const std::exception &) {
    std::cout << engine::serialize_response(engine::error_response(
                     "unavailable", "unavailable", "unavailable",
                     observer::engine_id, observer::version, "internal.failure",
                     "bounded observer failed"))
              << '\n';
    return 1;
  }
}
