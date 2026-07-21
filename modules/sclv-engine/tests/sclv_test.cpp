#include "provider.hpp"
#include "sclv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <filesystem>
#include <iostream>
#include <stdexcept>
#include <string>
#include <utility>

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
        test_recovery();
        fs::current_path(previous);
        std::cout << "SCLV engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "SCLV engine test failure: " << error.what() << '\n';
        return 1;
    }
}
