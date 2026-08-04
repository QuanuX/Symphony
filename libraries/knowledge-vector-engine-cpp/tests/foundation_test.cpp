#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <map>
#include <sstream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace fs = std::filesystem;
using namespace symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-kve-test-XXXXXX").string();
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

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

template <typename Function>
void require_error(Function&& function, const std::string& code) {
    try {
        function();
    } catch (const Error& error) {
        require(error.code() == code, "expected " + code + ", got " + error.code());
        return;
    }
    throw std::runtime_error("expected Error with code " + code);
}

std::string request_json(std::int64_t deadline) {
    return Json{
        {"protocol", process_protocol_v1},
        {"request_id", "request-1"},
        {"correlation_id", "correlation-1"},
        {"operation", "inspect"},
        {"target_engine", "symphony-test"},
        {"deadline_unix_ms", deadline},
        {"payload", Json::object()},
    }.dump();
}

void test_digest() {
    require(
        sha256_hex("") == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
        "empty SHA-256 golden mismatch");
    require(
        sha256_hex("abc") == "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
        "abc SHA-256 golden mismatch");
}

void test_json_and_protocol() {
    const std::int64_t now = 1700000000000LL;
    const auto request = parse_request(request_json(now + 1000), "symphony-test", now);
    require(request.operation == "inspect", "request operation mismatch");

    require_error([&] {
        static_cast<void>(parse_bounded_json("{\"a\":1,\"a\":2}", Limits::max_request_bytes));
    }, "json.duplicate_key");
    const auto nested_siblings = parse_bounded_json(
        "{\"items\":[{\"name\":\"first\"},{\"name\":\"second\"}]}",
        Limits::max_request_bytes);
    require(nested_siblings.at("items").size() == 2U, "valid sibling objects were rejected");
    require_error([&] {
        static_cast<void>(parse_bounded_json("{\"value\":1.5}", Limits::max_request_bytes));
    }, "json.float_prohibited");
    require_error([&] {
        static_cast<void>(parse_bounded_json("{\"value\":9007199254740992}", Limits::max_request_bytes));
    }, "json.integer_out_of_range");
    require_error([&] {
        static_cast<void>(parse_bounded_json("{} trailing", Limits::max_request_bytes));
    }, "json.invalid");
    std::string invalid_utf8 = "{\"value\":\"";
    invalid_utf8.push_back(static_cast<char>(0xff));
    invalid_utf8 += "\"}";
    require_error([&] {
        static_cast<void>(parse_bounded_json(invalid_utf8, Limits::max_request_bytes));
    }, "json.invalid");

    std::string excessive_depth;
    for (std::size_t index = 0; index < Limits::max_json_depth + 2U; ++index) {
        excessive_depth += "[";
    }
    excessive_depth += "null";
    for (std::size_t index = 0; index < Limits::max_json_depth + 2U; ++index) {
        excessive_depth += "]";
    }
    require_error([&] {
        static_cast<void>(parse_bounded_json(excessive_depth, Limits::max_request_bytes));
    }, "json.depth_exceeded");

    const auto oversized_string = Json{{"value", std::string(Limits::max_string_bytes + 1U, 'x')}}.dump();
    require_error([&] {
        static_cast<void>(parse_bounded_json(oversized_string, Limits::max_request_bytes));
    }, "json.string_too_large");

    auto unknown = Json::parse(request_json(now + 1000));
    unknown["unknown"] = true;
    require_error([&] {
        static_cast<void>(parse_request(unknown.dump(), "symphony-test", now));
    }, "request.field_set");

    require_error([&] {
        static_cast<void>(parse_request(request_json(now), "symphony-test", now));
    }, "request.deadline_expired");
    require_error([&] {
        static_cast<void>(parse_request(
            request_json(now + Limits::max_deadline_ahead_ms + 1), "symphony-test", now));
    }, "request.deadline_too_far");
    require_error([&] {
        static_cast<void>(parse_request(request_json(now + 1000), "another-engine", now));
    }, "engine.target_mismatch");

    const auto encoded = serialize_response(success_response(
        request, "symphony-test", "0.1.0-dev", Json{{"ready", true}}));
    const auto parsed = Json::parse(encoded);
    require(parsed.at("outcome") == "ok", "response outcome mismatch");
    const auto digest = parsed.at("response_digest").get<std::string>();
    auto without_digest = parsed;
    without_digest.erase("response_digest");
    require(digest == tagged_sha256(without_digest.dump()), "response digest mismatch");
    require_error([&] {
        static_cast<void>(serialize_response(success_response(
            request, "symphony-test", "0.1.0-dev", Json{{"float", 1.5}})));
    }, "response.invalid");

    std::istringstream oversized(std::string(Limits::max_request_bytes + 1U, 'x'));
    require_error([&] {
        static_cast<void>(read_bounded(oversized, Limits::max_request_bytes));
    }, "input.too_large");
}

void test_paths_and_snapshots() {
    require(is_safe_relative_path("knowledge/INTENT.md"), "expected safe path");
    require(!is_safe_relative_path("../INTENT.md"), "traversal accepted");
    require(!is_safe_relative_path("/tmp/file"), "absolute path accepted");
    require(!is_safe_relative_path("knowledge//INTENT.md"), "empty component accepted");
    require(!is_safe_relative_path("knowledge\\INTENT.md"), "backslash accepted");

    TemporaryDirectory temporary;
    fs::create_directories(temporary.path() / "knowledge");
    {
        std::ofstream output(temporary.path() / "knowledge" / "INTENT.md", std::ios::binary);
        output << "canonical\n";
    }
    const auto first = snapshot_files(temporary.path(), {"knowledge/INTENT.md"});
    const auto second = snapshot_files(temporary.path(), {"knowledge/INTENT.md"});
    require(first.digest == second.digest, "snapshot is not deterministic");
    require(first.files.size() == 1U, "snapshot file count mismatch");

    require_error([&] {
        static_cast<void>(snapshot_files(
            temporary.path(), {"knowledge/INTENT.md", "knowledge/INTENT.md"}));
    }, "snapshot.duplicate_path");

    fs::create_directory_symlink(temporary.path() / "knowledge", temporary.path() / "linked");
    require_error([&] {
        static_cast<void>(read_regular_file_no_follow(
            temporary.path(), "linked/INTENT.md", Limits::max_snapshot_file_bytes));
    }, "path.component_unsafe");
    fs::create_symlink(temporary.path() / "knowledge" / "INTENT.md", temporary.path() / "final-link");
    require_error([&] {
        static_cast<void>(read_regular_file_no_follow(
            temporary.path(), "final-link", Limits::max_snapshot_file_bytes));
    }, "path.file_unreadable");

    const auto root_link = temporary.path().parent_path() / (temporary.path().filename().string() + "-link");
    fs::create_directory_symlink(temporary.path(), root_link);
    require_error([&] {
        static_cast<void>(read_regular_file_no_follow(
            root_link, "knowledge/INTENT.md", Limits::max_snapshot_file_bytes));
    }, "path.root_unsafe");
    fs::remove(root_link);

    require_error([&] {
        static_cast<void>(snapshot_files(
            temporary.path(), {"knowledge/INTENT.md"}, unix_time_ms() - 1));
    }, "request.deadline_expired");
}

void test_schema_documents(const fs::path& repository_root) {
    const std::map<std::string, std::string> expected = {
        {"knowledge/schemas/v1/engine-process-request.schema.json", "urn:symphony:knowledge:engine-process:request:v1"},
        {"knowledge/schemas/v1/engine-process-response.schema.json", "urn:symphony:knowledge:engine-process:response:v1"},
        {"knowledge/schemas/v1/engine-descriptor.schema.json", "urn:symphony:knowledge:engine-descriptor:v1"},
        {"knowledge/schemas/v1/install-receipt.schema.json", "urn:symphony:knowledge:install-receipt:v1"},
        {"knowledge/schemas/v1/engine-binding-registry.schema.json", "urn:symphony:knowledge:engine-binding-registry:v1"},
        {"knowledge/schemas/v1/reconciliation-command.schema.json", "urn:symphony:knowledge:reconciliation-command:v1"},
        {"knowledge/schemas/v1/reconciliation-head.schema.json", "urn:symphony:knowledge:reconciliation-head:v1"},
        {"knowledge/schemas/v1/reconciliation-journal.schema.json", "urn:symphony:knowledge:reconciliation-journal:v1"},
        {"knowledge/schemas/v1/reconciliation-result.schema.json", "urn:symphony:knowledge:reconciliation-result:v1"},
        {"knowledge/schemas/v1/session-command.schema.json", "urn:symphony:knowledge:session-command:v1"},
        {"knowledge/schemas/v1/session-head.schema.json", "urn:symphony:knowledge:session-head:v1"},
        {"knowledge/schemas/v1/session-journal.schema.json", "urn:symphony:knowledge:session-journal:v1"},
        {"knowledge/schemas/v1/session-result.schema.json", "urn:symphony:knowledge:session-result:v1"},
        {"knowledge/schemas/v1/session-transition-result.schema.json", "urn:symphony:knowledge:session-transition-result:v1"},
        {"knowledge/schemas/v1/lifecycle-desired-state.schema.json", "urn:symphony:knowledge:lifecycle-desired-state:v1"},
        {"knowledge/schemas/v1/lifecycle-observation.schema.json", "urn:symphony:knowledge:lifecycle-observation:v1"},
        {"knowledge/schemas/v1/lifecycle-plan-command.schema.json", "urn:symphony:knowledge:lifecycle-plan-command:v1"},
        {"knowledge/schemas/v1/lifecycle-plan.schema.json", "urn:symphony:knowledge:lifecycle-plan:v1"},
        {"knowledge/schemas/v1/lifecycle-applied-state.schema.json", "urn:symphony:knowledge:lifecycle-applied-state:v1"},
        {"knowledge/schemas/v1/lifecycle-boot-journal.schema.json", "urn:symphony:knowledge:lifecycle-boot-journal:v1"},
        {"knowledge/schemas/v1/lifecycle-boot-head.schema.json", "urn:symphony:knowledge:lifecycle-boot-head:v1"},
        {"knowledge/schemas/v2/install-receipt.schema.json", "urn:symphony:knowledge:install-receipt:v2"},
    };
    for (const auto& [relative_path, identifier] : expected) {
        std::ifstream input(repository_root / relative_path, std::ios::binary);
        require(input.good(), "schema could not be opened: " + relative_path);
        const auto contents = read_bounded(input, Limits::max_request_bytes);
        const auto schema = parse_bounded_json(contents, Limits::max_request_bytes);
        require(schema.is_object(), "schema is not an object: " + relative_path);
        require(schema.at("$schema") == "https://json-schema.org/draft/2020-12/schema", "schema dialect mismatch");
        require(schema.at("$id") == identifier, "schema identifier mismatch: " + relative_path);
        require(schema.at("type") == "object", "schema root type mismatch: " + relative_path);
        require(schema.at("additionalProperties") == false, "schema root is not closed: " + relative_path);
    }

    const auto load_schema = [&](const std::string& relative_path) {
        std::ifstream input(repository_root / relative_path, std::ios::binary);
        require(input.good(), "schema could not be opened: " + relative_path);
        return parse_bounded_json(read_bounded(input, Limits::max_request_bytes), Limits::max_request_bytes);
    };

    const auto plan = load_schema("knowledge/schemas/v1/lifecycle-plan.schema.json");
    const auto& scheduler = plan.at("$defs").at("scheduler").at("properties");
    require(scheduler.at("algorithm").at("const") == "dependency_ready_set_v1", "lifecycle scheduler drift");
    require(scheduler.at("dynamic_replanning").at("const") == true, "dynamic replanning must be required");
    require(scheduler.at("directionality").at("const") == "forward_and_inverse", "two-way action drift");
    require(scheduler.at("max_actions").at("const") == 4096, "lifecycle action bound drift");
    require(scheduler.at("max_replans_per_transaction").at("const") == 256, "lifecycle replan bound drift");
    require(scheduler.at("max_attempts_per_action").at("const") == 8, "lifecycle attempt bound drift");
    require(
        scheduler.at("safety_phase_order").at("const") == Json::array({
            "lock", "observe", "authorize", "compare_and_swap", "act", "verify", "audit"}),
        "lifecycle safety phase order drift");
    const auto& action_required = plan.at("$defs").at("action").at("required");
    require(std::find(action_required.begin(), action_required.end(), "target_state_digest") != action_required.end(),
            "lifecycle action target-state binding drift");
    require(std::find(action_required.begin(), action_required.end(), "target_receptor_id") != action_required.end(),
            "lifecycle action receptor binding drift");
    const auto& plan_required = plan.at("required");
    require(std::find(plan_required.begin(), plan_required.end(), "advisories") != plan_required.end(),
            "lifecycle noncritical-dependency advisory drift");

    const auto observation = load_schema("knowledge/schemas/v1/lifecycle-observation.schema.json");
    const auto& observed_required = observation.at("$defs").at("component").at("required");
    require(std::find(observed_required.begin(), observed_required.end(), "receptor_id") != observed_required.end(),
            "lifecycle observation receptor identity drift");

    const auto applied = load_schema("knowledge/schemas/v1/lifecycle-applied-state.schema.json");
    const auto& applied_required = applied.at("$defs").at("component").at("required");
    require(std::find(applied_required.begin(), applied_required.end(), "receptor_id") != applied_required.end(),
            "lifecycle applied-state receptor identity drift");

    const auto receipt_v2 = load_schema("knowledge/schemas/v2/install-receipt.schema.json");
    const auto& receipt_properties = receipt_v2.at("properties");
    require(!receipt_properties.contains("state"), "receipt v2 must not own mutable state");
    require(!receipt_properties.contains("active"), "receipt v2 must not own activation state");
    require(!receipt_properties.contains("default_receptor"), "receipt v2 must not own receptor selection");
    require(receipt_properties.at("files").at("items").at("$ref") == "#/$defs/file", "receipt v2 file evidence drift");
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root argument is required");
        }
        test_digest();
        test_json_and_protocol();
        test_paths_and_snapshots();
        test_schema_documents(fs::path(argv[1]));
        std::cout << "knowledge vector engine foundation tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
