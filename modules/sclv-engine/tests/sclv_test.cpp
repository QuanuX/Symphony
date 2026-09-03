#include "provider.hpp"
#include "sclv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>
#include <utility>
#include <vector>

namespace fs = std::filesystem;
namespace sclv = symphony::knowledge::sclv;
namespace provider = symphony::knowledge::sclv::provider;
namespace engine = symphony::knowledge::engine;

namespace {

void require(bool condition, const std::string& message) {
    if (!condition) throw std::runtime_error(message);
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

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-sclv-retirement-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) throw std::runtime_error("mkdtemp failed");
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

void write_file(const fs::path& root, const std::string& relative, const std::string& contents) {
    const auto path = root / relative;
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) throw std::runtime_error("could not create SCLV test fixture: " + relative);
    output << contents;
}

void create_retirement_fixture(
    const fs::path& root,
    const std::string& record_markdown) {
    const std::vector<std::string> contract_paths = {
        "knowledge/SPEC.md",
        "knowledge/schemas/v1/provider-evidence.schema.json",
        "knowledge/sclv/INTENT.md",
        "knowledge/sclv/MANIFEST.md",
        "knowledge/sclv/RECOVERY.md",
        "knowledge/sclv/SKILL.md",
        "knowledge/sclv/SPEC.md",
        "knowledge/sclv/schemas/v3/record.schema.json",
        "knowledge/sclv/schemas/v3/proposal-input.schema.json",
        "knowledge/sclv/schemas/v3/recovery-input.schema.json",
        "knowledge/sclv/schemas/v3/check-result.schema.json",
        "knowledge/sclv/schemas/v3/projection.schema.json",
        "knowledge/sclv/templates/v3/record.md",
    };
    for (const auto& path : contract_paths) write_file(root, path, "fixture\n");
    write_file(root, "knowledge/sclv/CHANGELOG.md", "# SCLV retirement fixture\n\n" + record_markdown);
    write_file(
        root,
        "knowledge/skvi/INDEX.md",
        "# Current-only SKVI fixture\n\n- path: `knowledge/NAMESPACES.md`\n");
    write_file(root, "knowledge/NAMESPACES.md", "# Stable identity tombstone fixture\n");
}

bool contains_path(const engine::Json& files, const std::string& path) {
    return std::any_of(files.begin(), files.end(), [&](const engine::Json& file) {
        return file.at("path") == path;
    });
}

bool contains_evidence_code(const engine::Json& evidence, const std::string& code) {
    return std::any_of(evidence.begin(), evidence.end(), [&](const engine::Json& item) {
        return item.at("code") == code;
    });
}

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{
        "request-1", "correlation-1", std::move(operation), sclv::engine_id,
        engine::unix_time_ms() + 60000, std::move(payload),
    };
}

const std::string revision(40U, 'a');
const std::string tree_digest = engine::tagged_sha256("fixture tree");
const std::string ratification_digest = engine::tagged_sha256("fixture ratification");

engine::Json change_request() {
    return engine::Json{
        {"state", "not_applicable"}, {"provider", "not_applicable"},
        {"id", "not_applicable"}, {"reference", "not_applicable"},
        {"absence_reason", "test change was ratified without a forge change request"},
    };
}

engine::Json ratification() {
    return engine::Json{
        {"state", "asserted"}, {"subject", "fixture-owner"},
        {"effective_permission", "repository-transition-owner"},
        {"method", "airgap-declaration"}, {"evidence_reference", "fixture-ratification"},
        {"evidence_digest", ratification_digest}, {"absence_reason", "not_applicable"},
    };
}

engine::Json repository_evidence() {
    return engine::Json{
        {"revision_scheme", "git-sha1"}, {"revision_value", revision},
        {"tree_digest", tree_digest},
    };
}

engine::Json normalized_evidence() {
    return provider::normalize_airgap(engine::Json{
        {"observed_at", "2099-01-01T00:10:00Z"},
        {"source_reference", "fixture-airgap-record"},
        {"repository", repository_evidence()},
        {"change_request", change_request()},
        {"ratification", ratification()},
    });
}

engine::Json record(std::string disposition = "post_merge", std::string reason = "not_applicable") {
    return engine::Json{
        {"record_id", "SCLV-CHG-FIXTURE-0001"}, {"record_version", 3},
        {"title", "Fixture provider-neutral record"}, {"status", "canonical"},
        {"date", "2099-01-01"}, {"change_started_at", "2099-01-01T00:00:00Z"},
        {"change_completed_at", "2099-01-01T00:20:00Z"}, {"recorded_at", "2099-01-01T00:30:00Z"},
        {"recording_disposition", std::move(disposition)}, {"recovery_reason", std::move(reason)},
        {"change_type", "implementation_change"},
        {"change_request_state", "not_applicable"}, {"change_request_provider", "not_applicable"},
        {"change_request_id", "not_applicable"}, {"change_request_reference", "not_applicable"},
        {"change_request_absence_reason", "test change was ratified without a forge change request"},
        {"revision_scheme", "git-sha1"}, {"revision_value", revision}, {"tree_digest", tree_digest},
        {"ratification_subject", "fixture-owner"},
        {"ratification_permission", "repository-transition-owner"},
        {"ratification_method", "airgap-declaration"},
        {"ratification_evidence_reference", "fixture-ratification"},
        {"ratification_evidence_digest", ratification_digest},
        {"affected_surfaces", engine::Json::array({"README.md"})},
        {"skvi_references", engine::Json::array({"README.md"})},
        {"change_summary", "Exercises the exact SCLV v3 proposal boundary."},
        {"relationship_changes", "No relationship change."}, {"doctrine_changes", "No doctrine change."},
        {"compatibility_consequences", "No compatibility consequence."},
        {"publication_consequences", "No publication is authorized."},
        {"projection_consequences", "Projection remains derived and rebuildable."},
        {"evidence", engine::Json::array({"fixture evidence"})},
        {"non_authorizations", engine::Json::array({"canonical apply"})},
        {"notes", "Test-only proposal content."},
    };
}

engine::Json repository() {
    return engine::Json{
        {"repository_id", "fixture-repository"},
        {"revision", engine::Json{{"scheme", "git-sha1"}, {"value", revision}}},
        {"worktree_id", "fixture-worktree"}, {"tree_digest", tree_digest},
    };
}

engine::Json proposal_input(engine::Json value) {
    return engine::Json{
        {"repository", repository()}, {"session_ref", "session-1"}, {"context_ref", "context-1"},
        {"proposal_expires_at", "2099-01-01T01:00:00Z"}, {"record", std::move(value)},
        {"provider_evidence", engine::Json::array({normalized_evidence()})},
    };
}

void test_actual_repository(const fs::path& root) {
    fs::current_path(root);
    const auto descriptor = sclv::descriptor();
    require(descriptor.at("language") == "C++26", "engine language drifted");
    require(descriptor.at("thermal_path") == "freezing", "thermal classification drifted");
    require(descriptor.at("canonical_apply_enabled") == false, "apply became enabled");
    require(descriptor.at("session_mutation_enabled") == false, "session mutation became enabled");
    require(descriptor.at("network_listener") == false, "network listener became enabled");

    const auto inspected = sclv::handle_request(request("inspect", engine::Json::object()));
    require(inspected.at("read_only") == true, "inspect is not read-only");
    const auto checked = sclv::handle_request(request(
        "check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(checked.at("summary").at("state") == "valid", "actual SCLV ledger is invalid");
    require(checked.at("summary").at("violation") == 0, "actual SCLV ledger has violations");
    require(checked.at("records_checked").get<std::size_t>() >= 15U, "record coverage regressed");
    const auto digest = checked.at("ledger").at("digest");
    const auto matching = sclv::handle_request(request(
        "check", engine::Json{{"expected_ledger_digest", digest}}));
    require(matching.at("expected_ledger_matches") == true, "expected ledger state did not match");
    const auto stale = sclv::handle_request(request("check", engine::Json{
        {"expected_ledger_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
    }));
    require(stale.at("expected_ledger_matches") == false, "stale expected state was accepted");
    require(stale.at("summary").at("state") == "invalid", "stale expected state was not a violation");

    const auto first = sclv::handle_request(request("project", engine::Json{{"format", "json"}}));
    const auto second = sclv::handle_request(request("project", engine::Json{{"format", "json"}}));
    require(first == second, "projection is not deterministic");
    require(first.at("noncanonical") == true && first.at("rebuildable") == true, "projection authority drifted");
    require(first.at("record_count") == checked.at("records_checked"), "projection count mismatch");
    auto projection = first;
    const auto projection_digest = projection.at("projection_digest").get<std::string>();
    projection.erase("projection_digest");
    require(projection_digest == engine::tagged_sha256(projection.dump()), "projection digest mismatch");
}

void test_provider_and_proposal() {
    const auto evidence = normalized_evidence();
    provider::validate_evidence(evidence);
    require(evidence.at("provider_namespace") == "airgap", "air-gap namespace mismatch");
    auto damaged = evidence;
    damaged["source_reference"] = "changed";
    require_error([&] { provider::validate_evidence(damaged); }, "provider.digest_mismatch");

    auto build_version = evidence;
    build_version["adapter_version"] = "0.1.0+fixture";
    build_version.erase("evidence_digest");
    build_version["evidence_digest"] = engine::tagged_sha256(build_version.dump());
    provider::validate_evidence(build_version);

    auto invalid_version = evidence;
    invalid_version["adapter_version"] = "0.1.0_fixture";
    invalid_version.erase("evidence_digest");
    invalid_version["evidence_digest"] = engine::tagged_sha256(invalid_version.dump());
    require_error([&] { provider::validate_evidence(invalid_version); }, "provider.identity_invalid");

    const auto first = sclv::handle_request(request("propose", proposal_input(record())));
    const auto second = sclv::handle_request(request("propose", proposal_input(record())));
    require(first == second, "proposal is not deterministic");
    require(first.at("protocol") == "symphony.knowledge.proposal.v1", "proposal protocol mismatch");
    require(first.at("authority").at("caller_declared_operation") == true, "caller declaration is absent");
    require(first.at("authority").at("engine_decided_domain_truth") == false, "engine claimed domain-truth authority");
    require(first.at("authority").at("ratified") == false, "engine self-ratified");
    require(first.at("canonical_apply_enabled") == false, "proposal enabled apply");
    require(first.at("write_set").size() == 1U, "proposal write-set is not singular");
    require(first.at("write_set").at(0).at("target_path") == "knowledge/sclv/CHANGELOG.md", "proposal escaped ledger");
    require(first.at("operations").at(0).at("data").at("markdown").get<std::string>().starts_with(
        "- record_id: `SCLV-CHG-FIXTURE-0001`"), "canonical Markdown rendering mismatch");

    auto mismatched = proposal_input(record());
    mismatched["record"]["tree_digest"] = engine::tagged_sha256("wrong");
    require_error([&] {
        static_cast<void>(sclv::handle_request(request("propose", mismatched)));
    }, "proposal.evidence_mismatch");
}

void test_removal_and_retirement_semantics() {
    const std::string removed_surface = "modules/node-troll/INTENT.md";
    const std::string tombstone_surface = "knowledge/NAMESPACES.md";
    require(!fs::exists(removed_surface), "retired module prose unexpectedly remains present");
    require(fs::is_regular_file(tombstone_surface), "stable-identity tombstone surface is absent");

    auto retirement_record = record();
    retirement_record["record_id"] = "SCLV-CHG-FIXTURE-RETIREMENT";
    retirement_record["affected_surfaces"] = engine::Json::array({removed_surface});
    retirement_record["skvi_references"] = engine::Json::array({tombstone_surface});
    const auto proposal = sclv::handle_request(request(
        "propose", proposal_input(retirement_record)));
    require(!contains_path(proposal.at("read_set"), removed_surface),
        "historical affected surface became a current proposal read obligation");
    require(contains_path(proposal.at("read_set"), tombstone_surface),
        "surviving tombstone evidence is absent from the proposal read set");
    require(contains_evidence_code(
                proposal.at("validation"), "sclv.affected_surfaces.provenance") &&
            contains_evidence_code(
                proposal.at("validation"), "sclv.skvi_references.current"),
        "proposal did not report the historical/current reference split");

    auto stale_reference_record = retirement_record;
    stale_reference_record["record_id"] = "SCLV-CHG-FIXTURE-STALE-REFERENCE";
    stale_reference_record["skvi_references"] = engine::Json::array({removed_surface});
    require_error([&] {
        static_cast<void>(sclv::handle_request(request(
            "propose", proposal_input(stale_reference_record))));
    }, "proposal.skvi_reference");

    const auto record_markdown = proposal.at("operations").at(0).at("data").at("markdown").get<std::string>();
    TemporaryDirectory temporary;
    create_retirement_fixture(temporary.path(), record_markdown);
    CurrentDirectory current(temporary.path());
    const auto valid = sclv::handle_request(request(
        "check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(valid.at("summary").at("state") == "valid",
        "runtime rejected an absent historical affected surface with surviving current evidence");

    fs::remove(temporary.path() / tombstone_surface);
    const auto missing_current_evidence = sclv::handle_request(request(
        "check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(missing_current_evidence.at("summary").at("state") == "invalid",
        "runtime accepted a missing current SKVI reference");
    require(contains_evidence_code(
                missing_current_evidence.at("evidence"), "sclv.record.skvi_unavailable"),
        "missing current reference did not produce exact runtime evidence");

    write_file(
        temporary.path(),
        "knowledge/skvi/INDEX.md",
        "# Current-only SKVI fixture after retirement\n");
    const auto retired_evidence = sclv::handle_request(request(
        "check", engine::Json{{"expected_ledger_digest", nullptr}}));
    require(retired_evidence.at("summary").at("state") == "valid" &&
            retired_evidence.at("summary").at("warning") == 1,
        "later retirement of historical SKVI evidence invalidated the immutable ledger");
    require(contains_evidence_code(
                retired_evidence.at("evidence"), "sclv.record.skvi_historical"),
        "retired historical SKVI reference did not produce exact warning evidence");
}

engine::Json journal() {
    return engine::Json{
        {"format_version", 1}, {"session_id", "session-1"}, {"source_operation", "closure"},
        {"base_revision", engine::Json{{"scheme", "git-sha1"}, {"value", std::string(40U, 'b')}}},
        {"intended_surfaces", engine::Json::array({"README.md"})},
        {"started_at", "2026-07-21T15:00:00Z"}, {"known_change_request", change_request()},
        {"known_revision", engine::Json{{"scheme", "git-sha1"}, {"value", revision}}},
        {"local_state", "resumable"},
    };
}

void test_recovery() {
    const auto state = journal();
    const auto digest = engine::tagged_sha256(state.dump());
    const auto resumed = sclv::handle_request(request("recover", engine::Json{
        {"journal", state}, {"journal_digest", digest}, {"observed_state", "still_open"},
        {"proposal_input", nullptr}, {"recovery_reason", "review remains open"},
    }));
    require(resumed.at("action") == "resume", "open recovery did not resume");
    require(resumed.at("journal_mutated") == false, "engine mutated the journal");
    require(resumed.at("delete_recommended") == false, "open journal was marked deletable");

    require_error([&] {
        static_cast<void>(sclv::handle_request(request("recover", engine::Json{
            {"journal", state}, {"journal_digest", digest}, {"observed_state", "indeterminate"},
            {"proposal_input", nullptr}, {"recovery_reason", "provider state is unavailable"},
        })));
    }, "recovery.indeterminate");

    require_error([&] {
        static_cast<void>(sclv::handle_request(request("recover", engine::Json{
            {"journal", state}, {"journal_digest", digest}, {"observed_state", "merged_unrecorded"},
            {"proposal_input", engine::Json{{"record", engine::Json::object()}}},
            {"recovery_reason", "the record is structurally incomplete"},
        })));
    }, "recovery.late_record");

    const std::string reason = "the source change merged before the closure session resumed";
    const auto late = sclv::handle_request(request("recover", engine::Json{
        {"journal", state}, {"journal_digest", digest}, {"observed_state", "merged_unrecorded"},
        {"proposal_input", proposal_input(record("late_recovery", reason))}, {"recovery_reason", reason},
    }));
    require(late.at("action") == "propose_late_recovery", "late recovery did not propose forward correction");
    require(late.at("proposal").at("canonical_apply_enabled") == false, "recovery proposal enabled apply");
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) throw std::runtime_error("repository root argument is required");
        const auto previous = fs::current_path();
        test_actual_repository(fs::canonical(argv[1]));
        test_provider_and_proposal();
        test_removal_and_retirement_semantics();
        test_recovery();
        fs::current_path(previous);
        std::cout << "SCLV engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "SCLV engine test failure: " << error.what() << '\n';
        return 1;
    }
}
