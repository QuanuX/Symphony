#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <exception>
#include <iostream>
#include <string>

#ifndef SYMPHONY_SCV_DOMAIN
#error SYMPHONY_SCV_DOMAIN must identify the independently packaged domain
#endif

namespace engine = symphony::knowledge::engine;
namespace scv = symphony::knowledge::scv;
namespace {
const std::string domain = SYMPHONY_SCV_DOMAIN;
const std::string engine_id = "symphony-" + domain;
int emit_error(const engine::Error& error) {
    try {
        std::cout << engine::serialize_response(engine::error_response("unavailable", "unavailable", "unavailable",
            engine_id, scv::version, error.code(), error.what()));
    } catch (const std::exception&) { return 5; }
    return error.exit_status();
}
}
int main(int argc, char** argv) {
    try {
        if (argc == 2) {
            const std::string arg = argv[1];
            if (arg == "--version") { std::cout << engine_id << ' ' << scv::version << '\n'; return 0; }
            if (arg == "--descriptor") { std::cout << scv::descriptor(domain).dump() << '\n'; return 0; }
            if (arg == "--help") {
                std::cout << "Usage: " << engine_id << " [--help|--version|--descriptor]\n"
                    << "Reads one bounded symphony.knowledge.engine-process.v1 request from stdin.\n";
                return 0;
            }
            throw engine::Error("argument.unsupported", "unsupported argument", 2);
        }
        if (argc != 1) throw engine::Error("argument.count", "unexpected argument count", 2);
        auto request = engine::parse_request(engine::read_bounded(std::cin, engine::Limits::max_request_bytes), engine_id, engine::unix_time_ms());
        try {
            auto result = scv::handle_request(request, domain);
            std::cout << engine::serialize_response(engine::success_response(request, engine_id, scv::version, std::move(result)));
            return 0;
        } catch (const engine::Error& error) {
            std::cout << engine::serialize_response(engine::error_response(request.request_id, request.correlation_id,
                request.operation, engine_id, scv::version, error.code(), error.what()));
            return error.exit_status();
        }
    } catch (const engine::Error& error) { return emit_error(error); }
    catch (const std::exception&) { return emit_error(engine::Error("internal.failure", "bounded SCV processing failed", 5)); }
}
