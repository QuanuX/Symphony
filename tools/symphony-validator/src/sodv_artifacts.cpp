#include "sodv_artifacts.hpp"
#include "evidence.hpp"
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/path.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/knowledge/engine/temporal.hpp>
#include <algorithm>
#include <filesystem>
#include <map>
#include <regex>
#include <set>
#include <stdexcept>
#include <string>

namespace {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
constexpr const char* ledger = "knowledge/sodv/ARTIFACT-RELEASES.json";
void require(bool valid, const char* code) { if (!valid) throw std::runtime_error(code); }
void fields(const Json& object, std::initializer_list<const char*> names) {
    require(object.is_object() && object.size() == names.size(), "field_set");
    for (const auto name : names) require(object.contains(name), "field_set");
}
std::string text(const Json& value, std::size_t limit = 256) {
    require(value.is_string(), "string_type");
    const auto result = value.get<std::string>();
    require(!result.empty() && result.size() <= limit && std::all_of(result.begin(), result.end(),
        [](unsigned char c) { return c >= 0x20 && c <= 0x7e; }), "bounded_text");
    return result;
}
std::string pattern(const Json& value, const char* expression, std::size_t limit = 256) {
    const auto result = text(value, limit);
    require(std::regex_match(result, std::regex(expression)), "string_pattern");
    return result;
}
std::string sha(const Json& value) { return pattern(value, "[0-9a-f]{64}", 64); }
std::string token(const Json& value) { return pattern(value, "[A-Za-z0-9][A-Za-z0-9._:-]*"); }
std::string utc(const Json& value) {
    const auto result = text(value, 20); require(engine::is_utc_seconds(result), "timestamp"); return result;
}
std::uint64_t number(const Json& value, std::uint64_t maximum) {
    require(value.is_number_integer(), "integer_type");
    if (!value.is_number_unsigned()) require(value.get<std::int64_t>() >= 0, "negative_integer");
    const auto result = value.get<std::uint64_t>(); require(result <= maximum, "integer_bound"); return result;
}
void array(const Json& value, std::size_t minimum, std::size_t maximum) {
    require(value.is_array() && value.size() >= minimum && value.size() <= maximum, "array_bound");
}
std::string uri(const Json& value) {
    return pattern(value, "https://[^ /?#]+/[^ #]+", 4096);
}
std::string digest(Json value) {
    value.erase("record_sha256");
    return engine::tagged_sha256(value.dump()).substr(7);
}
std::string unit(const Json& value) {
    fields(value, {"unit_id", "artifact_kind", "component_id", "version", "source", "platform",
        "dependency_map_sha256", "dependencies", "artifact", "signature", "provenance_sha256", "inventory_sha256", "conda"});
    token(value.at("unit_id")); token(value.at("component_id"));
    // Package versions remain owner-defined, including conda's independent
    // version spelling; equality, not cross-ecosystem version ordering, governs.
    pattern(value.at("version"), "[A-Za-z0-9][A-Za-z0-9._+!-]*", 128);
    const auto kind = text(value.at("artifact_kind"));
    require(kind == "native_archive" || kind == "conda_package", "artifact_kind");
    const auto& source = value.at("source");
    fields(source, {"source_commit", "source_tree", "reference"});
    pattern(source.at("source_commit"), "[0-9a-f]{40}", 40);
    pattern(source.at("source_tree"), "[0-9a-f]{40}", 40); text(source.at("reference"), 4096);
    const auto& platform = value.at("platform");
    fields(platform, {"os", "architecture", "os_baseline", "abi"});
    const auto os = text(platform.at("os")); const auto arch = text(platform.at("architecture"));
    require(os == "linux" || os == "macos", "platform_os");
    require(arch == "x86_64" || arch == "aarch64", "platform_architecture");
    text(platform.at("os_baseline")); text(platform.at("abi"));
    sha(value.at("dependency_map_sha256")); sha(value.at("provenance_sha256")); sha(value.at("inventory_sha256"));
    array(value.at("dependencies"), 0, 256); std::set<std::string> dependencies;
    for (const auto& dependency : value.at("dependencies")) {
        fields(dependency, {"name", "version", "identity_sha256"});
        require(dependencies.insert(token(dependency.at("name"))).second, "duplicate_dependency");
        text(dependency.at("version"), 128); sha(dependency.at("identity_sha256"));
    }
    const auto& artifact = value.at("artifact");
    fields(artifact, {"filename", "sha256", "size", "immutable_uri"});
    pattern(artifact.at("filename"), "[A-Za-z0-9][A-Za-z0-9._+!-]*", 255);
    sha(artifact.at("sha256")); require(number(artifact.at("size"), 9007199254740991ULL) > 0, "artifact_size");
    uri(artifact.at("immutable_uri"));
    const auto& signature = value.at("signature");
    fields(signature, {"scheme", "namespace", "signer_identity", "key_id", "signature_sha256", "signed_payload_sha256"});
    text(signature.at("signer_identity"));
    require(signature.at("scheme") == "sshsig" && signature.at("namespace") == "symphony-release", "signature_scheme");
    pattern(signature.at("key_id"), "SHA256:[A-Za-z0-9+/]{43}", 50);
    sha(signature.at("signature_sha256")); sha(signature.at("signed_payload_sha256"));
    require(signature.at("signed_payload_sha256") == value.at("inventory_sha256"), "signature_subject");
    const auto& conda = value.at("conda");
    if (kind == "native_archive") require(conda.is_null(), "native_conda_metadata");
    else {
        fields(conda, {"name", "subdir", "build", "build_number"});
        pattern(conda.at("name"), "[a-z0-9][a-z0-9._-]*"); token(conda.at("build"));
        number(conda.at("build_number"), 9007199254740991ULL);
        const auto subdir = os == "macos" ? (arch == "aarch64" ? "osx-arm64" : "osx-64") :
            (arch == "aarch64" ? "linux-aarch64" : "linux-64");
        require(conda.at("subdir") == subdir, "conda_target");
        const auto filename = conda.at("name").get<std::string>() + "-" + value.at("version").get<std::string>() + "-" + conda.at("build").get<std::string>() + ".conda";
        require(artifact.at("filename") == filename, "conda_filename");
    }
    // Package coordinates cannot be re-authorized with different bytes/source.
    if (kind == "conda_package") return Json{{"kind", kind}, {"name", conda.at("name")}, {"version", value.at("version")},
        {"subdir", conda.at("subdir")}, {"build", conda.at("build")}}.dump();
    return Json{{"kind", kind}, {"component", value.at("component_id")}, {"version", value.at("version")},
        {"platform", platform}, {"conda", conda}}.dump();
}
void validate(const Json& document, SodvArtifactCheckResult& result) {
    fields(document, {"protocol", "format_version", "records"});
    require(document.at("protocol") == "symphony.sodv.artifact-release-ledger.v1" && number(document.at("format_version"), 1) == 1, "protocol");
    array(document.at("records"), 0, 4096);
    std::map<std::string, Json> authorizations;
    std::set<std::string> ids, completed, coordinates, locations;
    Json previous = nullptr; std::string previous_time;
    for (const auto& record : document.at("records")) {
        require(record.is_object() && record.contains("record_type"), "record_type");
        const auto type = text(record.at("record_type"));
        if (type == "authorization") fields(record, {"record_id", "record_type", "recorded_at", "recorded_by", "previous_record_sha256", "record_sha256", "authority", "units"});
        else {
            require(type == "completion", "record_type");
            fields(record, {"record_id", "record_type", "recorded_at", "recorded_by", "previous_record_sha256", "record_sha256", "authorization_id", "authorization_sha256", "authorization_merge", "published_at", "observations"});
        }
        const auto id = pattern(record.at("record_id"), "SODV-ART-[A-Z0-9][A-Z0-9._:-]*", 105);
        require(ids.insert(id).second, "duplicate_record"); text(record.at("recorded_by"));
        const auto recorded = utc(record.at("recorded_at")); require(recorded >= previous_time, "record_order");
        require(record.at("previous_record_sha256") == previous, "record_chain");
        require(sha(record.at("record_sha256")) == digest(record), "record_digest");
        if (type == "authorization") {
            const auto& authority = record.at("authority");
            fields(authority, {"identity", "publisher_identity", "evidence_sha256"});
            text(authority.at("identity")); text(authority.at("publisher_identity")); sha(authority.at("evidence_sha256"));
            array(record.at("units"), 1, 128); std::set<std::string> units;
            for (const auto& value : record.at("units")) {
                const auto coordinate = unit(value);
                require(units.insert(text(value.at("unit_id"))).second, "duplicate_unit");
                require(coordinates.insert(coordinate).second, "duplicate_coordinate");
                require(locations.insert(uri(value.at("artifact").at("immutable_uri"))).second, "duplicate_location");
                require(value.at("signature").at("signer_identity") == authority.at("publisher_identity"), "publisher_signer_identity");
            }
            authorizations.emplace(id, record);
        } else {
            const auto auth_id = text(record.at("authorization_id"), 105);
            require(authorizations.contains(auth_id), "authorization_missing");
            require(completed.insert(auth_id).second, "duplicate_completion");
            const auto& auth = authorizations.at(auth_id);
            require(sha(record.at("authorization_sha256")) == text(auth.at("record_sha256")), "authorization_digest");
            const auto& merge = record.at("authorization_merge");
            fields(merge, {"source_commit", "merged_at", "evidence_sha256"});
            pattern(merge.at("source_commit"), "[0-9a-f]{40}", 40); sha(merge.at("evidence_sha256"));
            const auto merged = utc(merge.at("merged_at")); const auto published = utc(record.at("published_at"));
            require(merged >= auth.at("recorded_at").get<std::string>() && published >= merged && recorded >= published, "publication_before_authorization");
            const auto& observations = record.at("observations"); array(observations, 1, 128);
            require(observations.size() == auth.at("units").size(), "completion_unit_set");
            std::set<std::string> observed;
            for (const auto& observation : observations) {
                fields(observation, {"unit_id", "artifact_sha256", "signature_sha256", "provenance_sha256", "inventory_sha256", "download_uri", "downloaded_bytes", "retrieved_at", "verification"});
                const auto unit_id = token(observation.at("unit_id")); require(observed.insert(unit_id).second, "duplicate_observation");
                const auto found = std::find_if(auth.at("units").begin(), auth.at("units").end(), [&](const auto& v) { return v.at("unit_id") == unit_id; });
                require(found != auth.at("units").end(), "unauthorized_unit");
                const auto& expected = *found;
                require(sha(observation.at("artifact_sha256")) == text(expected.at("artifact").at("sha256")) &&
                    sha(observation.at("signature_sha256")) == text(expected.at("signature").at("signature_sha256")) &&
                    sha(observation.at("provenance_sha256")) == text(expected.at("provenance_sha256")) &&
                    sha(observation.at("inventory_sha256")) == text(expected.at("inventory_sha256")), "completion_digest_mismatch");
                require(uri(observation.at("download_uri")) == text(expected.at("artifact").at("immutable_uri")) &&
                    number(observation.at("downloaded_bytes"), 9007199254740991ULL) == number(expected.at("artifact").at("size"), 9007199254740991ULL), "download_identity");
                const auto retrieved = utc(observation.at("retrieved_at")); require(retrieved >= published && retrieved <= recorded, "verification_time");
                const auto& verification = observation.at("verification");
                fields(verification, {"identity", "signer_identity", "key_id", "allowed_signers_sha256", "independent_of_publisher", "fresh_download", "checksum_match", "signature_verified", "provenance_verified", "inventory_verified", "evidence_sha256"});
                require(text(verification.at("identity")) != text(auth.at("authority").at("publisher_identity")), "publisher_is_verifier");
                require(text(verification.at("signer_identity")) == text(expected.at("signature").at("signer_identity")) &&
                    text(verification.at("key_id"), 50) == text(expected.at("signature").at("key_id")), "verification_signer");
                sha(verification.at("allowed_signers_sha256"));
                for (const auto field : {"independent_of_publisher", "fresh_download", "checksum_match", "signature_verified", "provenance_verified", "inventory_verified"})
                    require(verification.at(field).is_boolean() && verification.at(field) == true, "verification_incomplete");
                sha(verification.at("evidence_sha256"));
            }
            ++result.completed_transactions;
        }
        previous = record.at("record_sha256"); previous_time = recorded; ++result.records_checked;
    }
}
}

SodvArtifactCheckResult check_sodv_artifacts(const std::string& repository) {
    SodvArtifactCheckResult result;
    try {
        const auto root = std::filesystem::absolute(repository).lexically_normal();
        if (!std::filesystem::exists(root / ledger) && !std::filesystem::exists(root / "go.work")) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sodv.artifacts.fixture_not_applicable", "legacy minimal validator fixture"));
            return result;
        }
        const auto raw = engine::read_regular_file_no_follow(root, ledger, 4U * 1024U * 1024U);
        validate(engine::parse_bounded_json(raw, 4U * 1024U * 1024U), result);
        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sodv.artifacts.consistent",
            "records=" + std::to_string(result.records_checked) + " completions=" + std::to_string(result.completed_transactions) + " external_evidence_not_authenticated=true"));
    } catch (const std::exception& error) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sodv.artifacts.invalid", error.what()));
    }
    return result;
}
