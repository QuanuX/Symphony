#include "skvi.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <iterator>
#include <map>
#include <sstream>
#include <stdexcept>
#include <string>
#include <unistd.h>
#include <utility>
#include <vector>

namespace fs = std::filesystem;
namespace skvi = symphony::knowledge::skvi;
namespace engine = symphony::knowledge::engine;

namespace {

constexpr std::size_t expanded_fixture_entries = 2500;

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-skvi-test-XXXXXX").string();
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
        skvi::engine_id,
        engine::unix_time_ms() + 60000,
        std::move(payload),
    };
}

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("could not create fixture file: " + path.string());
    }
    output << contents;
}

const std::vector<std::string> required_paths = {
    "README.md",
    "INTENT.md",
    "go.work",
    "knowledge/INTENT.md",
    "knowledge/MANIFEST.md",
    "knowledge/SKILL.md",
    "knowledge/SPEC.md",
    "knowledge/skvi/INDEX.md",
    "knowledge/skvi/INTENT.md",
    "knowledge/skvi/MANIFEST.md",
    "knowledge/skvi/SKILL.md",
    "knowledge/skvi/SPEC.md",
};

std::string fixture_entry(const std::string& path, const std::string& title = "Fixture surface") {
    return "- path: `" + path + "`\n"
           "- title: " + title + "\n"
           "- surface_type: canonical fixture\n"
           "- truth_role: deterministic test truth\n"
           "- owner: fixture owner\n"
           "- scope: bounded test scope\n"
           "- relationships: none\n"
           "- consumers: SKVI tests\n"
           "- deferred_projections: none\n"
           "- notes: test-only canonical entry\n"
           "- status: canonical\n\n";
}

void create_fixture(const fs::path& root) {
    std::string index = "# Fixture SKVI Index\n\n";
    for (const auto& path : required_paths) {
        if (path != "knowledge/skvi/INDEX.md") {
            write_file(root / path, "fixture\n");
        }
        index += fixture_entry(path);
    }
    write_file(
        root / "knowledge/MANIFEST.md",
        "# Fixture Knowledge Manifest\n\n"
        "## Canonical Surfaces\n\n"
        "- `knowledge/INTENT.md`\n"
        "- `knowledge/MANIFEST.md`\n"
        "- `knowledge/SKILL.md`\n"
        "- `knowledge/SPEC.md`\n\n"
        "## Subordinate Manifests\n\n"
        "- `knowledge/skvi/MANIFEST.md`\n");
    write_file(
        root / "knowledge/skvi/MANIFEST.md",
        "# Fixture SKVI Manifest\n\n"
        "## Canonical Surfaces\n\n"
        "- `knowledge/skvi/INDEX.md`\n"
        "- `knowledge/skvi/INTENT.md`\n"
        "- `knowledge/skvi/MANIFEST.md`\n"
        "- `knowledge/skvi/SKILL.md`\n"
        "- `knowledge/skvi/SPEC.md`\n");
    write_file(root / "knowledge/skvi/INDEX.md", index);
    write_file(root / "candidate.md", "candidate\n");
}

engine::Json fixture_repository() {
    return engine::Json{
        {"repository_id", "fixture-repository"},
        {"revision", engine::Json{{"scheme", "git"}, {"value", "fixture-revision"}}},
        {"worktree_id", "fixture-worktree"},
        {"tree_digest", "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
    };
}

engine::Json candidate_entry() {
    return engine::Json{
        {"path", "candidate.md"},
        {"title", "Candidate"},
        {"surface_type", "canonical fixture"},
        {"truth_role", "candidate truth"},
        {"owner", "fixture owner"},
        {"scope", "bounded test scope"},
        {"relationships", "none"},
        {"consumers", "SKVI tests"},
        {"deferred_projections", "none"},
        {"notes", "caller-declared candidate"},
        {"status", "canonical"},
    };
}

engine::Json proposal_payload(engine::Json operation) {
    return engine::Json{
        {"repository", fixture_repository()},
        {"session_ref", "session-1"},
        {"context_ref", "context-1"},
        {"created_at", "2026-07-21T12:00:00Z"},
        {"expires_at", "2026-07-21T12:05:00Z"},
        {"operation", std::move(operation)},
    };
}

void test_descriptor_and_actual_repository(const fs::path& repository_root) {
    const auto descriptor = skvi::descriptor();
    require(descriptor.at("engine_id") == skvi::engine_id, "descriptor engine mismatch");
    require(descriptor.at("limits").at("json_values") == skvi::max_json_values,
        "descriptor JSON value ceiling mismatch");
    require(descriptor.at("limits").at("snapshot_files") == engine::Limits::max_snapshot_files,
        "descriptor snapshot ceiling mismatch");
    require(engine::Limits::max_manifest_files == 512U,
        "owner-manifest ceiling mismatch");
    require(descriptor.at("canonical_apply_enabled") == false, "apply must remain disabled");
    require(descriptor.at("session_mutation_enabled") == false, "session mutation must remain disabled");
    require(descriptor.at("network_listener") == false, "network listener must remain disabled");

    CurrentDirectory current(repository_root);
    const auto inspect = skvi::handle_request(request("inspect", engine::Json::object()));
    require(inspect.at("engine_decides_membership") == false, "engine claimed membership authority");

    const auto check = skvi::handle_request(request(
        "check", engine::Json{{"expected_index_digest", nullptr}}));
    require(check.at("summary").at("state") == "valid", "actual SKVI index is invalid");
    require(check.at("summary").at("violation") == 0, "actual index has violations");
    require(check.at("entries_checked").get<std::size_t>() > 0U,
        "actual index contains no entries");
    const auto current_project_request = request("project", engine::Json{{"format", "json"}});
    const auto current_projection = skvi::handle_request(current_project_request);
    require(current_projection.at("entry_count") == check.at("entries_checked"),
        "actual projection entry count does not match check evidence");
    const auto current_response = engine::serialize_response(engine::success_response(
        current_project_request, skvi::engine_id, skvi::engine_version, current_projection),
        skvi::max_json_values);
    require(current_response.size() <= (engine::Limits::max_response_bytes * 3U) / 4U,
        "actual projection has less than one MiB of response headroom");
    const auto index_digest = check.at("index").at("digest").get<std::string>();
    const auto matching = skvi::handle_request(request(
        "check", engine::Json{{"expected_index_digest", index_digest}}));
    require(matching.at("expected_index_matches") == true, "matching expected index digest failed");
    const auto stale = skvi::handle_request(request("check", engine::Json{
        {"expected_index_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
    }));
    require(stale.at("expected_index_matches") == false, "stale expected index digest was accepted");
    require(stale.at("summary").at("state") == "invalid", "stale expected state was not a violation");

    const auto first = skvi::handle_request(request("project", engine::Json{{"format", "json"}}));
    const auto second = skvi::handle_request(request("project", engine::Json{{"format", "json"}}));
    require(first == second, "projection is not deterministic");
    require(first.at("noncanonical") == true, "projection claimed canonical status");
    require(first.at("rebuildable") == true, "projection is not rebuildable");
    require(first.at("entry_count") == check.at("entries_checked"), "projection count mismatch");
    auto projection_without_digest = first;
    const auto projection_digest = projection_without_digest.at("projection_digest").get<std::string>();
    projection_without_digest.erase("projection_digest");
    require(
        projection_digest == engine::tagged_sha256(projection_without_digest.dump()),
        "projection digest mismatch");
    for (const auto& projected_entry : first.at("entries")) {
        auto entry_without_digest = projected_entry;
        const auto entry_digest = entry_without_digest.at("entry_digest").get<std::string>();
        entry_without_digest.erase("entry_digest");
        require(entry_digest == engine::tagged_sha256(entry_without_digest.dump()), "entry digest mismatch");
    }
}

void test_proposal_semantics() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());

    const auto operation = engine::Json{
        {"type", "add_entry"},
        {"target_path", "candidate.md"},
        {"expected_entry_digest", nullptr},
        {"entry", candidate_entry()},
    };
    const auto first = skvi::handle_request(request("propose", proposal_payload(operation)));
    const auto second = skvi::handle_request(request("propose", proposal_payload(operation)));
    require(first == second, "proposal is not deterministic for declared inputs");
    require(first.at("protocol") == "symphony.knowledge.proposal.v1", "proposal protocol mismatch");
    require(first.at("authority").at("caller_declared_operation") == true, "caller declaration absent");
    require(first.at("authority").at("engine_decided_domain_truth") == false, "engine decided domain truth");
    require(first.at("authority").at("ratified") == false, "proposal self-ratified");
    require(first.at("canonical_apply_enabled") == false, "proposal enabled apply");
    require(first.at("operations").size() == 1U, "proposal operation count mismatch");
    require(first.at("write_set").at(0).at("target_path") == "knowledge/skvi/INDEX.md", "write target escaped SKVI");
    auto proposal_without_digest = first;
    const auto proposal_digest = proposal_without_digest.at("proposal_digest").get<std::string>();
    proposal_without_digest.erase("proposal_digest");
    require(
        proposal_digest == engine::tagged_sha256(proposal_without_digest.dump()),
        "proposal digest mismatch");

    const auto projection = skvi::handle_request(request("project", engine::Json{{"format", "json"}}));
    const auto current_entry = projection.at("entries").at(0);
    const auto target_path = current_entry.at("path").get<std::string>();
    require_error([&] {
        static_cast<void>(skvi::handle_request(request("propose", proposal_payload(engine::Json{
            {"type", "replace_entry"},
            {"target_path", target_path},
            {"expected_entry_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
            {"entry", candidate_entry()},
        }))));
    }, "proposal.expected_state_mismatch");

    require_error([&] {
        static_cast<void>(skvi::handle_request(request("propose", proposal_payload(engine::Json{
            {"type", "add_entry"},
            {"target_path", "candidate.md"},
            {"expected_entry_digest", nullptr},
            {"entry", nullptr},
        }))));
    }, "proposal.entry_required");
}

void test_drift_and_no_follow() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        std::ofstream output(temporary.path() / "knowledge/skvi/INDEX.md", std::ios::app | std::ios::binary);
        output << fixture_entry("README.md", "Duplicate");
        output.close();
        CurrentDirectory current(temporary.path());
        const auto check = skvi::handle_request(request(
            "check", engine::Json{{"expected_index_digest", nullptr}}));
        require(check.at("summary").at("state") == "invalid", "duplicate index path was accepted");
        require(check.at("summary").at("violation").get<std::size_t>() > 0U, "duplicate produced no violation");
        require_error([&] {
            static_cast<void>(skvi::handle_request(request("project", engine::Json{{"format", "json"}})));
        }, "skvi.index_invalid");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "README.md");
        fs::create_symlink(temporary.path() / "candidate.md", temporary.path() / "README.md");
        CurrentDirectory current(temporary.path());
        const auto check = skvi::handle_request(request(
            "check", engine::Json{{"expected_index_digest", nullptr}}));
        require(check.at("summary").at("state") == "invalid", "symlinked indexed file was accepted");
    }
}

void test_bounds_and_reserved_operations() {
    require_error([&] {
        static_cast<void>(skvi::handle_request(request("inspect", engine::Json{{"extra", true}})));
    }, "payload.field_set");
    require_error([&] {
        static_cast<void>(skvi::handle_request(request("project", engine::Json{{"format", "jsonl"}})));
    }, "payload.unsupported_format");
    for (const std::string operation : {"apply", "begin", "close", "recover"}) {
        require_error([&] {
            static_cast<void>(skvi::handle_request(request(operation, engine::Json::object())));
        }, "operation.unsupported");
    }
}

void test_process_envelope_capacity() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    std::ofstream index(temporary.path() / "knowledge/skvi/INDEX.md", std::ios::app | std::ios::binary);
    for (std::size_t number = required_paths.size(); number < expanded_fixture_entries; ++number) {
        std::ostringstream path;
        path << "fixture/surface-" << number << ".md";
        write_file(temporary.path() / path.str(), "fixture\n");
        index << fixture_entry(path.str());
    }
    index.close();

    CurrentDirectory current(temporary.path());
    const auto check = skvi::handle_request(request(
        "check", engine::Json{{"expected_index_digest", nullptr}}));
    require(check.at("entries_checked") == expanded_fixture_entries, "expanded check entry count mismatch");
    require(check.at("summary").at("state") == "valid", "expanded index is invalid");
    const auto project_request = request("project", engine::Json{{"format", "json"}});
    const auto projection = skvi::handle_request(project_request);
    require(projection.at("entry_count") == expanded_fixture_entries, "expanded projection entry count mismatch");
    require_error([&] {
        static_cast<void>(engine::serialize_response(engine::success_response(
            project_request, skvi::engine_id, skvi::engine_version, projection)));
    }, "response.invalid");
    const auto response = engine::serialize_response(engine::success_response(
        project_request, skvi::engine_id, skvi::engine_version, projection),
        skvi::max_json_values);
    require(!response.empty(), "expanded projection did not fit the common response envelope");
    require(engine::parse_bounded_json(
        response, engine::Limits::max_response_bytes, skvi::max_json_values)
            .at("result").at("entry_count") == expanded_fixture_entries,
        "expanded projection did not fit the SKVI JSON event envelope");
    // A larger valid index is still checkable even when a full projection cannot
    // fit the existing process JSON budget. No new count ceiling replaces 2,048.
    constexpr std::size_t larger_fixture_entries = 3000;
    std::ofstream more(temporary.path() / "knowledge/skvi/INDEX.md", std::ios::app | std::ios::binary);
    for (std::size_t number = expanded_fixture_entries; number < larger_fixture_entries; ++number) {
        const auto path = "fixture/surface-" + std::to_string(number) + ".md";
        write_file(temporary.path() / path, "fixture\n");
        more << fixture_entry(path);
    }
    more.close();
    const auto larger_check = skvi::handle_request(request(
        "check", engine::Json{{"expected_index_digest", nullptr}}));
    require(larger_check.at("entries_checked") == larger_fixture_entries,
        "larger check entry count mismatch");
    require(larger_check.at("summary").at("state") == "valid", "larger index is invalid");
    const auto larger_check_request = request("check", engine::Json{{"expected_index_digest", nullptr}});
    static_cast<void>(engine::serialize_response(engine::success_response(
        larger_check_request, skvi::engine_id, skvi::engine_version, larger_check), skvi::max_json_values));
    static_cast<void>(skvi::handle_request(request("propose", proposal_payload(engine::Json{
        {"type", "add_entry"}, {"target_path", "candidate.md"},
        {"expected_entry_digest", nullptr}, {"entry", candidate_entry()},
    }))));
    const auto larger_project_request = request("project", engine::Json{{"format", "json"}});
    const auto larger_projection = skvi::handle_request(larger_project_request);
    require(larger_projection.at("entry_count") == larger_fixture_entries,
        "larger projection entry count mismatch");
    require_error([&] {
        static_cast<void>(engine::serialize_response(engine::success_response(
            larger_project_request, skvi::engine_id, skvi::engine_version, larger_projection),
            skvi::max_json_values));
    }, "response.invalid");

    // Escaping can exceed response bytes even when the source index fits its
    // own byte bound. The response guard remains independent of index admission.
    std::ifstream input(temporary.path() / "knowledge/skvi/INDEX.md", std::ios::binary);
    std::string escaped_index((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
    input.close();
    const std::string original_notes = "- notes: test-only canonical entry";
    const std::string escaped_notes = "- notes: " + std::string(800, '\\');
    std::size_t position = 0;
    while ((position = escaped_index.find(original_notes, position)) != std::string::npos) {
        escaped_index.replace(position, original_notes.size(), escaped_notes);
        position += escaped_notes.size();
    }
    require(escaped_index.size() < engine::Limits::max_snapshot_file_bytes,
        "escaped fixture index must fit the input byte bound");
    write_file(temporary.path() / "knowledge/skvi/INDEX.md", escaped_index);
    const auto escaped_project_request = request("project", engine::Json{{"format", "json"}});
    const auto escaped_projection = skvi::handle_request(escaped_project_request);
    require_error([&] {
        static_cast<void>(engine::serialize_response(engine::success_response(
            escaped_project_request, skvi::engine_id, skvi::engine_version, escaped_projection),
            skvi::max_json_values));
    }, "response.too_large");

    write_file(temporary.path() / "knowledge/skvi/INDEX.md",
        std::string(engine::Limits::max_snapshot_file_bytes + 1U, 'x'));
    require_error([&] {
        static_cast<void>(skvi::handle_request(request(
            "check", engine::Json{{"expected_index_digest", nullptr}})));
    }, "path.file_too_large");
}

void test_schema_documents(const fs::path& repository_root) {
    const std::map<std::string, std::string> expected = {
        {"knowledge/schemas/v1/proposal.schema.json", "urn:symphony:knowledge:proposal:v1"},
        {"knowledge/skvi/schemas/v1/entry.schema.json", "urn:symphony:skvi:entry:v1"},
        {"knowledge/skvi/schemas/v1/operation-payload.schema.json", "urn:symphony:skvi:operation-payload:v1"},
        {"knowledge/skvi/schemas/v1/check-result.schema.json", "urn:symphony:skvi:check-result:v1"},
        {"knowledge/skvi/schemas/v1/projection.schema.json", "urn:symphony:skvi:projection:v1"},
        {"knowledge/skvi/schemas/v2/check-result.schema.json", "urn:symphony:skvi:check-result:v2"},
        {"knowledge/skvi/schemas/v2/projection.schema.json", "urn:symphony:skvi:projection:v2"},
    };
    for (const auto& [relative_path, identifier] : expected) {
        std::ifstream input(repository_root / relative_path, std::ios::binary);
        require(input.good(), "schema could not be opened: " + relative_path);
        const auto document = engine::parse_bounded_json(
            engine::read_bounded(input, engine::Limits::max_request_bytes),
            engine::Limits::max_request_bytes);
        require(document.at("$schema") == "https://json-schema.org/draft/2020-12/schema", "schema dialect mismatch");
        require(document.at("$id") == identifier, "schema identifier mismatch: " + relative_path);
        require(document.at("type") == "object", "schema root type mismatch: " + relative_path);
        require(document.at("additionalProperties") == false, "schema root is not closed: " + relative_path);
        if (identifier == "urn:symphony:skvi:check-result:v1" ||
            identifier == "urn:symphony:skvi:check-result:v2") {
            const auto& count = document.at("properties").at("entries_checked");
            if (identifier.ends_with(":v1")) {
                require(count.at("maximum") == 1024U, "v1 check entry ceiling changed");
            } else {
                require(!count.contains("maximum"), "v2 check retains an entry-count ceiling");
            }
            require(document.at("$defs").at("snapshot").at("properties").at("files").at("maxItems") ==
                    engine::Limits::max_snapshot_files,
                "check schema snapshot ceiling mismatch");
        } else if (identifier == "urn:symphony:skvi:projection:v1" ||
                   identifier == "urn:symphony:skvi:projection:v2") {
            const auto& count = document.at("properties").at("entry_count");
            const auto& entries = document.at("properties").at("entries");
            if (identifier.ends_with(":v1")) {
                require(count.at("maximum") == 1024U && entries.at("maxItems") == 1024U,
                    "v1 projection entry ceilings changed");
            } else {
                require(!count.contains("maximum") && !entries.contains("maxItems"),
                    "v2 projection retains an entry-count ceiling");
            }
            require(document.at("$defs").at("snapshot").at("properties").at("files").at("maxItems") ==
                    engine::Limits::max_snapshot_files,
                "projection schema snapshot ceiling mismatch");
        }
    }
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root argument is required");
        }
        const fs::path repository_root = fs::canonical(argv[1]);
        test_descriptor_and_actual_repository(repository_root);
        test_proposal_semantics();
        test_drift_and_no_follow();
        test_bounds_and_reserved_operations();
        test_process_envelope_capacity();
        test_schema_documents(repository_root);
        std::cout << "SKVI engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
