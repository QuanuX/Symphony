#include "skvi.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <exception>
#include <iostream>
#include <string>
#include <utility>

namespace skvi = symphony::knowledge::skvi;
namespace engine = symphony::knowledge::engine;

namespace {

int emit_error(const engine::Error& error) {
    try {
        std::cout << engine::serialize_response(engine::error_response(
            "unavailable",
            "unavailable",
            "unavailable",
            skvi::engine_id,
            skvi::engine_version,
            error.code(),
            error.what()));
    } catch (const std::exception&) {
        return 5;
    }
    return error.exit_status();
}

}

int main(int argc, char** argv) {
    if (argc == 2) {
        const std::string argument = argv[1];
        if (argument == "--help") {
            std::cout << "Usage: symphony-skvi [--help|--version|--descriptor]\n"
                         "Without arguments, reads one bounded process request from standard input.\n";
            return 0;
        }
        if (argument == "--version") {
            std::cout << skvi::engine_id << ' ' << skvi::engine_version << '\n';
            return 0;
        }
        if (argument == "--descriptor") {
            std::cout << skvi::descriptor().dump() << '\n';
            return 0;
        }
        return emit_error(engine::Error("argument.unsupported", "unsupported argument", 2));
    }
    if (argc != 1) {
        return emit_error(engine::Error("argument.count", "unexpected argument count", 2));
    }

    try {
        const auto input = engine::read_bounded(std::cin, engine::Limits::max_request_bytes);
        const auto request = engine::parse_request(input, skvi::engine_id, engine::unix_time_ms());
        try {
            auto result = skvi::handle_request(request);
            std::cout << engine::serialize_response(engine::success_response(
                request, skvi::engine_id, skvi::engine_version, std::move(result)));
            return 0;
        } catch (const engine::Error& error) {
            std::cout << engine::serialize_response(engine::error_response(
                request.request_id,
                request.correlation_id,
                request.operation,
                skvi::engine_id,
                skvi::engine_version,
                error.code(),
                error.what()));
            return error.exit_status();
        }
    } catch (const engine::Error& error) {
        return emit_error(error);
    } catch (const std::exception&) {
        return emit_error(engine::Error("internal.failure", "bounded request processing failed", 5));
    }
}
