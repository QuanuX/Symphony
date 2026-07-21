#include "provider.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <exception>
#include <iostream>
#include <string>
#include <utility>

namespace provider = symphony::knowledge::sclv::provider;
namespace engine = symphony::knowledge::engine;

namespace {
constexpr const char* adapter_id = "symphony-sclv-evidence-airgap";

int emit_error(const engine::Error& error) {
    try {
        std::cout << engine::serialize_response(engine::error_response(
            "unavailable", "unavailable", "unavailable", adapter_id,
            provider::adapter_version, error.code(), error.what()));
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
            std::cout << "Usage: symphony-sclv-evidence-airgap [--help|--version|--descriptor]\n"
                         "Without arguments, reads one bounded normalize request from standard input.\n";
            return 0;
        }
        if (argument == "--version") {
            std::cout << adapter_id << ' ' << provider::adapter_version << '\n';
            return 0;
        }
        if (argument == "--descriptor") {
            std::cout << provider::descriptor(adapter_id).dump() << '\n';
            return 0;
        }
        return emit_error(engine::Error("argument.unsupported", "unsupported argument", 2));
    }
    if (argc != 1) return emit_error(engine::Error("argument.count", "unexpected argument count", 2));
    try {
        const auto input = engine::read_bounded(std::cin, engine::Limits::max_request_bytes);
        const auto request = engine::parse_request(input, adapter_id, engine::unix_time_ms());
        try {
            if (request.operation != "normalize") {
                throw engine::Error("operation.unsupported", "operation is unsupported", 4);
            }
            auto result = provider::normalize_airgap(request.payload);
            std::cout << engine::serialize_response(engine::success_response(
                request, adapter_id, provider::adapter_version, std::move(result)));
            return 0;
        } catch (const engine::Error& error) {
            std::cout << engine::serialize_response(engine::error_response(
                request.request_id, request.correlation_id, request.operation, adapter_id,
                provider::adapter_version, error.code(), error.what()));
            return error.exit_status();
        }
    } catch (const engine::Error& error) {
        return emit_error(error);
    } catch (const std::exception&) {
        return emit_error(engine::Error("internal.failure", "bounded request processing failed", 5));
    }
}
