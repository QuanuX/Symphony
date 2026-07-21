#include "coordinator.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace fs = std::filesystem;
namespace session = symphony::knowledge::session;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-session-test-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) {
            throw std::runtime_error("mkdtemp failed");
        }
        path_ = result;
    }
    ~TemporaryDirectory() {
        std::error_code ignored;
        fs::remove_all(path_, ignored);
    }
    TemporaryDirectory(const TemporaryDirectory&) = delete;
    TemporaryDirectory& operator=(const TemporaryDirectory&) = delete;
    [[nodiscard]] const fs::path& path() const { return path_; }

private:
    fs::path path_;
};

class CurrentDirectory final {
public:
    explicit CurrentDirectory(const fs::path& path) : previous_(fs::current_path()) {
        fs::current_path(path);
    }
    ~CurrentDirectory() {
        std::error_code ignored;
        fs::current_path(previous_, ignored);
    }
    CurrentDirectory(const CurrentDirectory&) = delete;
    CurrentDirectory& operator=(const CurrentDirectory&) = delete;

private:
    fs::path previous_;
};

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

template <typename Function>
void require_error(Function&& function, const std::string& code) {
    try {
        function();
    } catch (const engine::Error& error) {
        require(error.code() == code, "expected " + code + ", got " + error.code());
        return;
    }
    throw std::runtime_error("expected Error with code " + code);
}

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{
        "request-1",
        "correlation-1",
        std::move(operation),
        session::engine_id,
        engine::unix_time_ms() + 60000,
        std::move(payload),
    };
}

void test_descriptor_and_inspect() {
    const auto descriptor = session::descriptor();
    require(descriptor.at("engine_id") == session::engine_id, "descriptor engine mismatch");
    require(descriptor.at("canonical_apply_enabled") == false, "apply must remain disabled");
    require(descriptor.at("session_mutation_enabled") == false, "session mutation must remain disabled");
    require(descriptor.at("network_listener") == false, "network listener must remain disabled");

    const auto result = session::handle_request(request("inspect", engine::Json::object()));
    require(result.at("readiness") == "read_only_foundation", "inspect readiness mismatch");
    require(result.at("maestro_docking_enabled") == false, "docking must remain disabled");

    require_error([&] {
        static_cast<void>(session::handle_request(request("inspect", engine::Json{{"extra", true}})));
    }, "payload.field_set");
}

void test_check() {
    TemporaryDirectory temporary;
    {
        std::ofstream output(temporary.path() / "INTENT.md", std::ios::binary);
        output << "intent\n";
    }
    CurrentDirectory current(temporary.path());
    const auto first = session::handle_request(request("check", engine::Json{
        {"paths", engine::Json::array({"INTENT.md"})},
        {"expected_snapshot_digest", nullptr},
    }));
    const auto digest = first.at("snapshot").at("digest").get<std::string>();
    require(first.at("expected_snapshot_matches").is_null(), "unexpected expected-state result");
    require(first.at("read_only") == true, "check must be read-only");

    const auto second = session::handle_request(request("check", engine::Json{
        {"paths", engine::Json::array({"INTENT.md"})},
        {"expected_snapshot_digest", digest},
    }));
    require(second.at("expected_snapshot_matches") == true, "expected digest did not match");
    const auto mismatch = session::handle_request(request("check", engine::Json{
        {"paths", engine::Json::array({"INTENT.md"})},
        {"expected_snapshot_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
    }));
    require(mismatch.at("expected_snapshot_matches") == false, "stale expected digest was accepted");

    require_error([&] {
        static_cast<void>(session::handle_request(request("check", engine::Json{
            {"paths", engine::Json::array({"../INTENT.md"})},
            {"expected_snapshot_digest", nullptr},
        })));
    }, "path.unsafe");
    require_error([&] {
        static_cast<void>(session::handle_request(request("check", engine::Json{
            {"paths", engine::Json::array({"INTENT.md"})},
            {"expected_snapshot_digest", "sha256:gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"},
        })));
    }, "payload.invalid_expected_digest");
}

void test_reserved_operations() {
    for (const std::string operation : {"begin", "status", "checkpoint", "close", "recover", "apply"}) {
        require_error([&] {
            static_cast<void>(session::handle_request(request(operation, engine::Json::object())));
        }, "operation.unsupported");
    }
}

}

int main() {
    try {
        test_descriptor_and_inspect();
        test_check();
        test_reserved_operations();
        std::cout << "knowledge session coordinator tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
