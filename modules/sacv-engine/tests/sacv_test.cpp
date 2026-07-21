#include "sacv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <map>
#include <stdexcept>
#include <string>
#include <unistd.h>
#include <utility>
#include <vector>

namespace fs = std::filesystem;
namespace sacv = symphony::knowledge::sacv;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-sacv-test-XXXXXX").string();
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
    explicit CurrentDirectory(const fs::path& path) : previous_(fs::current_path()) { fs::current_path(path); }
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

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("could not create fixture: " + path.string());
    }
    output << contents;
}

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{"request-1", "correlation-1", std::move(operation), sacv::engine_id,
                           engine::unix_time_ms() + 60000, std::move(payload)};
}

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md", "knowledge/sacv/INTENT.md", "knowledge/sacv/MANIFEST.md",
    "knowledge/sacv/SKILL.md", "knowledge/sacv/SPEC.md",
    "knowledge/sacv/profiles/openapi-3.2.md", "knowledge/sacv/profiles/mintlify-publication.md",
    "knowledge/sacv/schemas/v1/MANIFEST.md", "knowledge/sacv/schemas/v1/registry-entry.schema.json",
    "knowledge/sacv/schemas/v1/check-result.schema.json", "knowledge/sacv/schemas/v1/diff-input.schema.json",
    "knowledge/sacv/schemas/v1/diff-result.schema.json", "knowledge/sacv/schemas/v1/proposal-input.schema.json",
    "knowledge/sacv/schemas/v1/projection.schema.json",
};

std::string openapi_document(const std::string& operation_id = "fixtureGet",
                             bool include_extra = false,
                             bool required_body = false) {
    const std::string extra = include_extra ? R"JSON(,
    "/extra": {
      "get": {
        "operationId": "fixtureExtra",
        "responses": {
          "200": {"description": "ok"},
          "400": {"description": "bad request"}
        }
      }
    })JSON" : "";
    const std::string body = required_body ? R"JSON(,
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object"}}}})JSON" : "";
    return R"JSON({
  "openapi": "3.2.0",
  "info": {"title": "Fixture API", "version": "1.0.0"},
  "x-symphony-security-profile": "knowledge/ssiag/SPEC.md",
  "security": [{"ssiag": []}],
  "paths": {
    "/fixture": {
      "get": {
        "operationId": ")JSON" + operation_id + R"JSON(" )JSON" + body + R"JSON(,
        "responses": {
          "200": {"description": "ok", "content": {"application/json": {"schema": {"type": "number", "minimum": 0.5}}}},
          "400": {"description": "bad request"}
        }
      }
    })JSON" + extra + R"JSON(
  },
  "components": {"securitySchemes": {"ssiag": {"type": "mutualTLS"}}}
})JSON";
}

std::string registry_entry(const std::string& path = "knowledge/fixture/api/fixture-v1.openapi.json") {
    return "- api_id: fixture-api\n"
           "- title: Fixture API\n"
           "- owner: `knowledge/fixture`\n"
           "- path: `" + path + "`\n"
           "- openapi: 3.2.0\n"
           "- api_version: 1.0.0\n"
           "- audience: local_internal\n"
           "- transport_profile: fixture-http\n"
           "- security_profile: knowledge/ssiag/SPEC.md\n"
           "- publication_state: internal_only\n"
           "- sdk_state: not_eligible\n"
           "- status: ratified\n"
           "- notes: deterministic fixture\n";
}

void create_fixture(const fs::path& root, bool registered = true) {
    for (const auto& path : contract_paths) {
        write_file(root / path, path.ends_with(".json") ? "{}\n" : "fixture\n");
    }
    const std::string api_path = "knowledge/fixture/api/fixture-v1.openapi.json";
    write_file(root / api_path, openapi_document());
    write_file(root / "knowledge/ssiag/SPEC.md", "fixture security profile\n");
    write_file(root / "knowledge/skvi/INDEX.md", "# Index\n\n- path: `" + api_path +
        "`\n\n- path: `knowledge/ssiag/SPEC.md`\n");
    if (registered) {
        write_file(root / "knowledge/sacv/REGISTRY.md",
                   "# Registry\n\n## Canonical Entries\n\n" + registry_entry());
    } else {
        write_file(root / "knowledge/sacv/REGISTRY.md",
                   "# Registry\n\n## Canonical Entries\n\nNone.\n");
    }
}

engine::Json repository_identity() {
    return engine::Json{{"repository_id", "fixture"},
        {"revision", engine::Json{{"scheme", "git"}, {"value", "fixture-revision"}}},
        {"worktree_id", "fixture-worktree"},
        {"tree_digest", "sha256:1111111111111111111111111111111111111111111111111111111111111111"}};
}

engine::Json entry_payload() {
    return engine::Json{{"api_id", "fixture-api"}, {"title", "Fixture API"},
        {"owner", "knowledge/fixture"}, {"path", "knowledge/fixture/api/fixture-v1.openapi.json"},
        {"openapi", "3.2.0"}, {"api_version", "1.0.0"}, {"audience", "local_internal"},
        {"transport_profile", "fixture-http"}, {"security_profile", "knowledge/ssiag/SPEC.md"},
        {"publication_state", "internal_only"}, {"sdk_state", "not_eligible"},
        {"status", "ratified"}, {"notes", "deterministic fixture"}};
}

void test_descriptor_and_actual_repository(const fs::path& repository_root) {
    const auto descriptor = sacv::descriptor();
    require(descriptor.at("openapi_target") == "3.2.0", "OpenAPI target mismatch");
    require(descriptor.at("parser_formats").at("json") == "implemented", "JSON parser absent");
    require(descriptor.at("parser_formats").at("yaml") == "fail_closed_unavailable", "YAML boundary unclear");
    require(descriptor.at("canonical_apply_enabled") == false, "apply enabled");
    require(descriptor.at("network_listener") == false, "listener enabled");

    CurrentDirectory current(repository_root);
    const auto inspect = sacv::handle_request(request("inspect", engine::Json::object()));
    require(inspect.at("engine_decides_ownership") == false, "engine claimed ownership authority");
    const auto check = sacv::handle_request(request("check", engine::Json{{"expected_registry_digest", nullptr}}));
    require(check.at("summary").at("state") == "valid", "empty canonical registry invalid");
    require(check.at("entries_checked") == 0U, "unexpected canonical API entry");
    const auto project = sacv::handle_request(request("project", engine::Json{{"format", "json"}}));
    require(project.at("entry_count") == 0U, "empty projection count mismatch");
    require(project.at("noncanonical") == true && project.at("rebuildable") == true,
            "projection authority escalated");
}

void test_registered_json_and_fail_closed_yaml() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());
    auto check = sacv::handle_request(request("check", engine::Json{{"expected_registry_digest", nullptr}}));
    require(check.at("summary").at("state") == "valid", "valid registered JSON rejected");
    require(check.at("entries_checked") == 1U && check.at("operations_checked") == 1U,
            "registered JSON counts mismatch");

    write_file(temporary.path() / "knowledge/fixture/api/fixture-v1.openapi.yaml",
               "openapi: 3.2.0\ninfo:\n  title: Fixture API\n  version: 1.0.0\npaths: {}\n");
    write_file(temporary.path() / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `knowledge/fixture/api/fixture-v1.openapi.yaml`\n\n"
               "- path: `knowledge/ssiag/SPEC.md`\n");
    write_file(temporary.path() / "knowledge/sacv/REGISTRY.md",
               "# Registry\n\n## Canonical Entries\n\n" +
               registry_entry("knowledge/fixture/api/fixture-v1.openapi.yaml"));
    check = sacv::handle_request(request("check", engine::Json{{"expected_registry_digest", nullptr}}));
    require(check.at("summary").at("state") == "invalid", "YAML was silently accepted");
    require(check.dump().find("sacv.document.parser_unavailable") != std::string::npos,
            "YAML parser-unavailable evidence missing");
}

void test_profile_violations_and_no_follow() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto unsafe = openapi_document();
        const auto marker = unsafe.find("\"paths\"");
        unsafe.insert(marker, "\"servers\":[{\"url\":\"https://prod.example.com\"}],\n");
        write_file(temporary.path() / "knowledge/fixture/api/fixture-v1.openapi.json", unsafe);
        CurrentDirectory current(temporary.path());
        const auto check = sacv::handle_request(request("check", engine::Json{{"expected_registry_digest", nullptr}}));
        require(check.at("summary").at("state") == "invalid", "internal server target accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "knowledge/fixture/api/fixture-v1.openapi.json");
        write_file(temporary.path() / "outside.openapi.json", openapi_document());
        fs::create_symlink(temporary.path() / "outside.openapi.json",
                           temporary.path() / "knowledge/fixture/api/fixture-v1.openapi.json");
        CurrentDirectory current(temporary.path());
        const auto check = sacv::handle_request(request("check", engine::Json{{"expected_registry_digest", nullptr}}));
        require(check.at("summary").at("state") == "invalid", "symlinked API document accepted");
    }
}

void test_diff_classification() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    const auto baseline_path = "knowledge/fixture/api/baseline.openapi.json";
    const auto candidate_path = "knowledge/fixture/api/candidate.openapi.json";
    const auto baseline = openapi_document();
    const auto additive = openapi_document("fixtureGet", true);
    write_file(temporary.path() / baseline_path, baseline);
    write_file(temporary.path() / candidate_path, additive);
    CurrentDirectory current(temporary.path());
    auto result = sacv::handle_request(request("diff", engine::Json{
        {"baseline_path", baseline_path}, {"baseline_digest", engine::tagged_sha256(baseline)},
        {"candidate_path", candidate_path}, {"candidate_digest", engine::tagged_sha256(additive)}}));
    require(result.at("state") == "compatible_additive", "additive operation misclassified");

    const auto breaking = openapi_document("renamedOperation");
    write_file(temporary.path() / candidate_path, breaking);
    result = sacv::handle_request(request("diff", engine::Json{
        {"baseline_path", baseline_path}, {"baseline_digest", engine::tagged_sha256(baseline)},
        {"candidate_path", candidate_path}, {"candidate_digest", engine::tagged_sha256(breaking)}}));
    require(result.at("state") == "breaking", "operationId change not breaking");

    auto malformed = openapi_document();
    const auto required = malformed.find("\"responses\"");
    malformed.insert(required, "\"requestBody\":{\"required\":\"yes\",\"content\":{\"application/json\":{}}},");
    write_file(temporary.path() / candidate_path, malformed);
    require_error([&] {
        static_cast<void>(sacv::handle_request(request("diff", engine::Json{
            {"baseline_path", baseline_path}, {"baseline_digest", engine::tagged_sha256(baseline)},
            {"candidate_path", candidate_path}, {"candidate_digest", engine::tagged_sha256(malformed)}})));
    }, "sacv.diff.document_invalid");

    auto external_reference = openapi_document();
    const auto local_schema = external_reference.find("\"type\": \"number\"");
    external_reference.replace(local_schema, std::string("\"type\": \"number\"").size(),
                               "\"$ref\": \"https://example.invalid/schema.json\"");
    write_file(temporary.path() / candidate_path, external_reference);
    require_error([&] {
        static_cast<void>(sacv::handle_request(request("diff", engine::Json{
            {"baseline_path", baseline_path}, {"baseline_digest", engine::tagged_sha256(baseline)},
            {"candidate_path", candidate_path},
            {"candidate_digest", engine::tagged_sha256(external_reference)}})));
    }, "sacv.diff.document_invalid");
}

void test_proposal_and_reserved_authority() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    CurrentDirectory current(temporary.path());
    std::ifstream registry("knowledge/sacv/REGISTRY.md", std::ios::binary);
    const std::string registry_contents((std::istreambuf_iterator<char>(registry)), std::istreambuf_iterator<char>());
    const auto payload = engine::Json{
        {"repository", repository_identity()}, {"session_ref", "session-1"},
        {"context_ref", "context-1"}, {"created_at", "2026-07-21T12:00:00Z"},
        {"expires_at", "2026-07-21T12:05:00Z"},
        {"operation", engine::Json{{"type", "register_contract"},
            {"expected_registry_digest", engine::tagged_sha256(registry_contents)},
            {"expected_entry_digest", nullptr}, {"entry", entry_payload()}}}};
    const auto first = sacv::handle_request(request("propose", payload));
    const auto second = sacv::handle_request(request("propose", payload));
    require(first == second, "proposal is not deterministic");
    require(first.at("authority").at("engine_decided_domain_truth") == false, "engine decided domain truth");
    require(first.at("authority").at("ratified") == false, "proposal self-ratified");
    require(first.at("canonical_apply_enabled") == false, "proposal enabled apply");
    require(first.at("write_set").at(0).at("target_path") == "knowledge/sacv/REGISTRY.md",
            "proposal escaped registry");
    auto invalid_payload = payload;
    invalid_payload["operation"]["entry"]["owner"] = "knowledge/other";
    require_error([&] {
        static_cast<void>(sacv::handle_request(request("propose", invalid_payload)));
    }, "payload.invalid_entry_ownership");
    for (const std::string operation : {"apply", "publish", "generate", "dock"}) {
        require_error([&] {
            static_cast<void>(sacv::handle_request(request(operation, engine::Json::object())));
        }, "operation.unsupported");
    }
}

void test_input_bounds() {
    require_error([&] {
        static_cast<void>(sacv::handle_request(request("inspect", engine::Json{{"extra", true}})));
    }, "payload.field_set");
    require_error([&] {
        static_cast<void>(sacv::handle_request(request("project", engine::Json{{"format", "yaml"}})));
    }, "payload.unsupported_format");
}

void test_schema_documents(const fs::path& repository_root) {
    const std::map<std::string, std::string> expected = {
        {"knowledge/sacv/schemas/v1/registry-entry.schema.json", "urn:symphony:sacv:registry-entry:v1"},
        {"knowledge/sacv/schemas/v1/check-result.schema.json", "urn:symphony:sacv:check-result:v1"},
        {"knowledge/sacv/schemas/v1/diff-input.schema.json", "urn:symphony:sacv:diff-input:v1"},
        {"knowledge/sacv/schemas/v1/diff-result.schema.json", "urn:symphony:sacv:diff-result:v1"},
        {"knowledge/sacv/schemas/v1/proposal-input.schema.json", "urn:symphony:sacv:proposal-input:v1"},
        {"knowledge/sacv/schemas/v1/projection.schema.json", "urn:symphony:sacv:projection:v1"},
    };
    for (const auto& [relative_path, identifier] : expected) {
        std::ifstream input(repository_root / relative_path, std::ios::binary);
        require(input.good(), "schema could not be opened: " + relative_path);
        const auto document = engine::parse_bounded_json(
            engine::read_bounded(input, engine::Limits::max_request_bytes),
            engine::Limits::max_request_bytes);
        require(document.at("$schema") == "https://json-schema.org/draft/2020-12/schema",
                "schema dialect mismatch: " + relative_path);
        require(document.at("$id") == identifier, "schema identifier mismatch: " + relative_path);
        require(document.at("type") == "object", "schema root type mismatch: " + relative_path);
        require(document.at("additionalProperties") == false,
                "schema root is not closed: " + relative_path);
    }
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root argument required");
        }
        test_descriptor_and_actual_repository(fs::canonical(argv[1]));
        test_registered_json_and_fail_closed_yaml();
        test_profile_violations_and_no_follow();
        test_diff_classification();
        test_proposal_and_reserved_authority();
        test_input_bounds();
        test_schema_documents(fs::canonical(argv[1]));
        std::cout << "SACV engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "SACV engine test failure: " << error.what() << '\n';
        return 1;
    }
}
