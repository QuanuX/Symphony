#include "sodv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>
#include <utility>
#include <vector>

namespace fs = std::filesystem;
namespace sodv = symphony::knowledge::sodv;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-sodv-test-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) { throw std::runtime_error("mkdtemp failed"); }
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
    if (!condition) { throw std::runtime_error(message); }
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
    if (!output.good()) { throw std::runtime_error("could not create fixture: " + path.string()); }
    output << contents;
}

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{"request-1", "correlation-1", std::move(operation), sodv::engine_id,
        engine::unix_time_ms() + 60000, std::move(payload)};
}

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md", "knowledge/sodv/INTENT.md", "knowledge/sodv/MANIFEST.md",
    "knowledge/sodv/SKILL.md", "knowledge/sodv/SPEC.md", "knowledge/sodv/RELEASES.md",
    "knowledge/sodv/schemas/v1/MANIFEST.md", "knowledge/sodv/schemas/v1/release-record-v2.schema.json",
    "knowledge/sodv/schemas/v1/observed-state.schema.json", "knowledge/sodv/schemas/v1/check-result.schema.json",
    "knowledge/sodv/schemas/v1/verify-result.schema.json", "knowledge/sodv/schemas/v1/proposal-input.schema.json",
    "knowledge/sodv/schemas/v1/recovery-input.schema.json", "knowledge/sodv/schemas/v1/recovery-result.schema.json",
    "knowledge/sodv/schemas/v1/projection.schema.json", "knowledge/skvi/INDEX.md",
};

void copy_contracts(const fs::path& repository, const fs::path& fixture) {
    for (const auto& path : contract_paths) {
        if (path == "knowledge/sodv/RELEASES.md") { continue; }
        fs::create_directories((fixture / path).parent_path());
        fs::copy_file(repository / path, fixture / path, fs::copy_options::overwrite_existing);
    }
}

std::string authorization_ledger() {
    return R"MD(# SODV Release Publication Ledger

## Release Records

- release_record_id: `SODV-REL-TEST1`
- record_version: `2`
- record_type: `authorization`
- status: `authorized`
- disposition: `test_authorization`
- recorded_at: `2026-07-20T00:00:00Z`
- recorded_by: `fixture-authority`
- subject_record_ids:
  - `not_applicable`
- publication_units:
  - unit_id: `fixture-module`
    artifact_kind: `go_module`
    coordinate: `example.com/fixture`
    version: `v1.0.0`
    tag: `fixture/v1.0.0`
    revision_scheme: `git-sha1`
    revision_value: `1111111111111111111111111111111111111111`
    source_reference: `fixture-source`
    tag_object: null
    content: null
    metadata: null
- purpose: `Exercise a provider-neutral release transaction.`
- evidence:
  - `fixture-evidence`
- non_authorizations:
  - `moving tags`
- notes: `Fixture authorization.`
)MD";
}

engine::Json repository_identity() {
    return engine::Json{{"repository_id", "fixture"},
        {"revision", engine::Json{{"scheme", "git"}, {"value", "fixture-revision"}}},
        {"worktree_id", "fixture-worktree"},
        {"tree_digest", "sha256:1111111111111111111111111111111111111111111111111111111111111111"}};
}

engine::Json observed_unit(const std::string& coordinate, const std::string& version, const std::string& tag,
                           const engine::Json& tag_object, const engine::Json& target,
                           const std::string& public_state, const engine::Json& content,
                           const engine::Json& metadata, char evidence = '2') {
    return engine::Json{{"coordinate", coordinate}, {"version", version}, {"tag", tag},
        {"tag_object", tag_object}, {"tag_target_revision", target}, {"public_state", public_state},
        {"public_content_digest", content}, {"public_metadata_digest", metadata},
        {"evidence_digest", "sha256:" + std::string(64, evidence)}};
}

engine::Json completed_observation() {
    auto units = engine::Json::array();
    units.push_back(observed_unit("github.com/QuanuX/Symphony/libraries/stav-protocol-go", "v0.2.0",
        "libraries/stav-protocol-go/v0.2.0", "f1274b6971941f8b60f991eb9b4422cc15703bb3",
        "55f8faf26f4f85213ac23cc1de7ba897b2129a4c", "resolved",
        "h1:DGVd771sqzeRpEkTUuuF+9TOK1JVQtyMh2GYR840g70=", "h1:kYeJSvzp7ezK+0CJzHD4v2euyRqXuAfXocYxRACrxoM="));
    units.push_back(observed_unit("github.com/QuanuX/Symphony/modules/stav-append-authority", "v0.1.0",
        "modules/stav-append-authority/v0.1.0", "dfa637080cf7e3b21cdd0b7e45fd5b0010a7fd5f",
        "55f8faf26f4f85213ac23cc1de7ba897b2129a4c", "resolved",
        "h1:iijcegHcZ8EXfKJ8v/ToZWvBuf2y81UDWpAjj+g8OpI=", "h1:pRWSy0nSQu5dYtiKpvTEmYTFrgf1O0bAqtmU3MDowlc=", '3'));
    units.push_back(observed_unit("github.com/QuanuX/Symphony/modules/stav-append-authority", "v0.2.0",
        "modules/stav-append-authority/v0.2.0", "aeb61f13c7e306a45818cde972307209d070dc28",
        "ed7484d70607aa96e64916dd4e59d3972a61980b", "resolved",
        "h1:DvWWrt7MbJFfEA/ROnTCDJYwoVWRgXSzy6IkTEpkMPI=", "h1:pRWSy0nSQu5dYtiKpvTEmYTFrgf1O0bAqtmU3MDowlc=", '4'));
    return engine::Json{{"authorization_record_id", "SODV-REL-001"},
        {"observed_at", "2026-07-20T01:00:00Z"}, {"source_reference", "caller-fixture"},
        {"units", std::move(units)}};
}

engine::Json fixture_observation(bool resolved) {
    auto units = engine::Json::array();
    if (resolved) {
        units.push_back(observed_unit("example.com/fixture", "v1.0.0", "fixture/v1.0.0",
            "2222222222222222222222222222222222222222", "1111111111111111111111111111111111111111",
            "resolved", "h1:fixture-content=", "h1:fixture-metadata="));
    } else {
        units.push_back(observed_unit("example.com/fixture", "v1.0.0", "fixture/v1.0.0",
            nullptr, nullptr, "not_observed", nullptr, nullptr));
    }
    return engine::Json{{"authorization_record_id", "SODV-REL-TEST1"},
        {"observed_at", "2026-07-20T01:00:00Z"}, {"source_reference", "caller-fixture"},
        {"units", std::move(units)}};
}

engine::Json proposed_authorization(const std::string& id = "SODV-REL-TEST2") {
    return engine::Json{{"release_record_id", id}, {"record_version", 2},
        {"record_type", "authorization"}, {"status", "authorized"}, {"disposition", "planned_release"},
        {"recorded_at", "2026-07-20T02:00:00Z"}, {"recorded_by", "fixture-authority"},
        {"subject_record_ids", engine::Json::array({"not_applicable"})},
        {"publication_units", engine::Json::array({engine::Json{{"unit_id", "fixture-module-two"},
            {"artifact_kind", "go_module"}, {"coordinate", "example.com/fixture-two"}, {"version", "v2.0.0"},
            {"tag", "fixture-two/v2.0.0"}, {"revision_scheme", "git-sha1"},
            {"revision_value", "3333333333333333333333333333333333333333"}, {"source_reference", "fixture-source"},
            {"tag_object", nullptr}, {"public_digests", engine::Json{{"content", nullptr}, {"metadata", nullptr}}}}})},
        {"purpose", "Authorize a second fixture release."}, {"evidence", engine::Json::array({"fixture-evidence"})},
        {"non_authorizations", engine::Json::array({"moving tags"})}, {"notes", "Fixture proposal."}};
}

engine::Json proposed_completion() {
    return engine::Json{{"release_record_id", "SODV-REL-TEST2"}, {"record_version", 2},
        {"record_type", "completion"}, {"status", "completed"}, {"disposition", "verified_completion"},
        {"recorded_at", "2026-07-20T02:00:00Z"}, {"recorded_by", "fixture-authority"},
        {"subject_record_ids", engine::Json::array({"SODV-REL-TEST1"})},
        {"publication_units", engine::Json::array({engine::Json{{"unit_id", "fixture-module"},
            {"artifact_kind", "go_module"}, {"coordinate", "example.com/fixture"}, {"version", "v1.0.0"},
            {"tag", "fixture/v1.0.0"}, {"revision_scheme", "git-sha1"},
            {"revision_value", "1111111111111111111111111111111111111111"}, {"source_reference", "fixture-source"},
            {"tag_object", "2222222222222222222222222222222222222222"},
            {"public_digests", engine::Json{{"content", "h1:fixture-content="}, {"metadata", "h1:fixture-metadata="}}}}})},
        {"purpose", "Complete the fixture release."}, {"evidence", engine::Json::array({"fixture-evidence"})},
        {"non_authorizations", engine::Json::array({"moving tags"})}, {"notes", "Fixture completion."}};
}

engine::Json proposal_input(const std::string& digest, engine::Json record, engine::Json observed) {
    return engine::Json{{"repository", repository_identity()}, {"session_ref", "session-1"},
        {"context_ref", nullptr}, {"proposal_expires_at", "2026-07-21T00:00:00Z"},
        {"expected_ledger_digest", digest}, {"record", std::move(record)}, {"observed_state", std::move(observed)}};
}

void test_actual_repository(const fs::path& repository) {
    CurrentDirectory current(repository);
    const auto descriptor = sodv::descriptor();
    require(descriptor.at("language") == "C++26", "language contract mismatch");
    require(descriptor.at("thermal_path") == "freezing", "thermal path mismatch");
    require(descriptor.at("network_access") == false, "network access enabled");
    require(descriptor.at("canonical_apply_enabled") == false, "canonical apply enabled");

    const auto check = sodv::handle_request(request("check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(check.at("summary").at("state") == "valid", "canonical ledger failed validation");
    require(check.at("records_checked") == 3U && check.at("transactions_checked") == 1U,
        "canonical release counts mismatch");

    const auto verify = sodv::handle_request(request("verify", completed_observation()));
    require(verify.at("verification_state") == "verified_completed", "canonical completion was not verified");
    require(verify.at("engine_declares_completion") == false, "engine claimed completion authority");

    auto mismatched = completed_observation();
    mismatched.at("units").at(0).at("tag_target_revision") = "9999999999999999999999999999999999999999";
    const auto blocked = sodv::handle_request(request("verify", mismatched));
    require(blocked.at("verification_state") == "blocked_mismatch", "tag mismatch did not fail closed");

    const auto project = sodv::handle_request(request("project", engine::Json{{"format", "json"}}));
    require(project.at("record_count") == 3U && project.at("transaction_count") == 1U,
        "projection counts mismatch");
    require(project.at("noncanonical") == true && project.at("rebuildable") == true,
        "projection claimed canonical authority");
}

void test_v2_proposals_and_recovery(const fs::path& repository) {
    TemporaryDirectory temporary;
    copy_contracts(repository, temporary.path());
    write_file(temporary.path() / "knowledge/sodv/RELEASES.md", authorization_ledger());
    CurrentDirectory current(temporary.path());
    const auto check = sodv::handle_request(request("check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(check.at("summary").at("state") == "valid", "v2 fixture ledger invalid: " + check.dump());
    const auto digest = check.at("ledger").at("digest").get<std::string>();

    const auto authorization = sodv::handle_request(request("propose",
        proposal_input(digest, proposed_authorization(), nullptr)));
    require(authorization.at("canonical_apply_enabled") == false, "proposal enabled apply");
    require(authorization.at("authority").at("ratified") == false, "proposal claimed ratification");
    require(authorization.at("operations").at(0).at("data").at("append_markdown").get<std::string>().find(
        "SODV-REL-TEST2") != std::string::npos, "deterministic append text missing");
    write_file(temporary.path() / "knowledge/sodv/RELEASES.md",
        authorization_ledger() + authorization.at("operations").at(0).at("data").at("append_markdown").get<std::string>());
    const auto reparsed = sodv::handle_request(request("check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(reparsed.at("summary").at("state") == "valid" && reparsed.at("records_checked") == 2U &&
        reparsed.at("transactions_checked") == 2U, "engine-rendered v2 record did not reparse exactly");
    write_file(temporary.path() / "knowledge/sodv/RELEASES.md", authorization_ledger());

    const auto completion = sodv::handle_request(request("propose",
        proposal_input(digest, proposed_completion(), fixture_observation(true))));
    require(completion.at("operations").at(0).at("data").at("verification").at("verification_state") ==
        "completion_candidate", "ready completion was not proposed");

    const auto journal = engine::Json{{"format_version", 1}, {"transaction_id", "transaction-1"},
        {"authorization_record_id", "SODV-REL-TEST1"}, {"started_at", "2026-07-20T00:00:00Z"},
        {"intended_tags", engine::Json::array({"fixture/v1.0.0"})}, {"local_state", "prepared"}};
    const auto recovery = sodv::handle_request(request("recover", engine::Json{{"journal", journal},
        {"journal_digest", engine::tagged_sha256(journal.dump())}, {"observed_state", fixture_observation(false)},
        {"proposal_input", nullptr}, {"recovery_reason", "session resumed after authentication"}}));
    require(recovery.at("action") == "resume_authorized_publication", "unpublished recovery action mismatch");
    require(recovery.at("journal_mutated") == false && recovery.at("delete_recommended") == false,
        "recovery mutated or discarded active journal");

    const auto ready = sodv::handle_request(request("recover", engine::Json{{"journal", journal},
        {"journal_digest", engine::tagged_sha256(journal.dump())}, {"observed_state", fixture_observation(true)},
        {"proposal_input", nullptr}, {"recovery_reason", "public evidence arrived"}}));
    require(ready.at("action") == "completion_proposal_required", "ready recovery bypassed proposal gate");

    require_error([&] { static_cast<void>(sodv::handle_request(request("propose",
        proposal_input("sha256:" + std::string(64, '9'), proposed_authorization(), nullptr)))); },
        "proposal.expected_state_mismatch");
}

void test_fail_closed_boundaries(const fs::path& repository) {
    require_error([&] { static_cast<void>(sodv::handle_request(request("apply", engine::Json::object()))); },
        "operation.unsupported");
    require_error([&] { static_cast<void>(sodv::handle_request(request("publish", engine::Json::object()))); },
        "operation.unsupported");
    require_error([&] { static_cast<void>(sodv::handle_request(request("tag", engine::Json::object()))); },
        "operation.unsupported");

    TemporaryDirectory temporary;
    copy_contracts(repository, temporary.path());
    write_file(temporary.path() / "outside.md", authorization_ledger());
    fs::create_directories(temporary.path() / "knowledge/sodv");
    fs::create_symlink(temporary.path() / "outside.md", temporary.path() / "knowledge/sodv/RELEASES.md");
    CurrentDirectory current(temporary.path());
    require_error([&] { static_cast<void>(sodv::handle_request(request("check",
        engine::Json{{"expected_ledger_digest", nullptr}}))); }, "path.file_unreadable");
}

void test_schema_documents(const fs::path& repository) {
    for (const auto& entry : fs::directory_iterator(repository / "knowledge/sodv/schemas/v1")) {
        if (entry.path().extension() != ".json") { continue; }
        std::ifstream input(entry.path(), std::ios::binary);
        const auto document = engine::Json::parse(input);
        require(document.at("$schema") == "https://json-schema.org/draft/2020-12/schema",
            "schema dialect mismatch: " + entry.path().string());
        require(document.at("$id").get<std::string>().starts_with("urn:symphony:sodv:"),
            "schema ID mismatch: " + entry.path().string());
        require(document.at("additionalProperties") == false,
            "schema root is not closed: " + entry.path().string());
    }
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) { throw std::runtime_error("repository root argument required"); }
        const auto repository = fs::canonical(argv[1]);
        test_actual_repository(repository);
        test_v2_proposals_and_recovery(repository);
        test_fail_closed_boundaries(repository);
        test_schema_documents(repository);
        std::cout << "sodv engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "sodv engine tests failed: " << error.what() << '\n';
        return 1;
    }
}
