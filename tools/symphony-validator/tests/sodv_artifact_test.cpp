#include "sodv_artifacts.hpp"
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/json.hpp>
#include <filesystem>
#include <fstream>
#include <functional>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace {
using Json = symphony::knowledge::engine::Json;
namespace fs = std::filesystem;
const std::string sha(64, 'a');
const std::string inventory_sha(64, 'b');
const std::string provenance_sha(64, 'c');
const std::string signature_sha(64, 'd');
void check(bool result, const std::string& message) { if (!result) throw std::runtime_error(message); }
void write(const fs::path& path, const std::string& bytes) {
    fs::create_directories(path.parent_path()); std::ofstream file(path); file << bytes;
    check(file.good(), "artifact fixture write failed");
}
struct Temporary {
    fs::path path;
    Temporary() {
        std::string value = (fs::canonical(fs::temp_directory_path()) / "sodv-artifact-XXXXXX").string();
        const auto result = ::mkdtemp(value.data()); check(result != nullptr, "artifact mkdtemp failed"); path = result;
        write(path / "go.work", "go 1.26.5\n");
    }
    ~Temporary() { std::error_code ignored; fs::remove_all(path, ignored); }
};
void seal(Json& record) { record.erase("record_sha256"); record["record_sha256"] = symphony::knowledge::engine::tagged_sha256(record.dump()).substr(7); }
void reseal(Json& ledger) {
    Json previous = nullptr;
    for (auto& record : ledger["records"]) { record["previous_record_sha256"] = previous; seal(record); previous = record["record_sha256"]; }
}
Json fixture() {
    Json unit{{"unit_id", "native-1"}, {"artifact_kind", "native_archive"}, {"component_id", "qxctl"}, {"version", "0.1.0-dev"},
        {"source", {{"source_commit", std::string(40, '1')}, {"source_tree", std::string(40, '2')}, {"reference", "fixture-only-source"}}},
        {"platform", {{"os", "linux"}, {"architecture", "aarch64"}, {"os_baseline", "fixture-os"}, {"abi", "fixture-abi"}}},
        {"dependency_map_sha256", sha}, {"dependencies", Json::array({{{"name", "fixture-lib"}, {"version", "1"}, {"identity_sha256", sha}}})},
        {"artifact", {{"filename", "qxctl.tar.gz"}, {"sha256", sha}, {"size", 123}, {"immutable_uri", "https://example.invalid/immutable/native-1"}}},
        {"signature", {{"scheme", "sshsig"}, {"namespace", "symphony-release"}, {"signer_identity", "fixture-publisher"}, {"key_id", "SHA256:" + std::string(43, 'A')}, {"signature_sha256", signature_sha}, {"signed_payload_sha256", inventory_sha}}},
        {"provenance_sha256", provenance_sha}, {"inventory_sha256", inventory_sha}, {"conda", nullptr}};
    auto conda = unit; conda["unit_id"] = "conda-1"; conda["artifact_kind"] = "conda_package"; conda["version"] = "0.1.0.dev0";
    conda["conda"] = {{"name", "symphony-qxctl"}, {"subdir", "linux-aarch64"}, {"build", "fixture_0"}, {"build_number", 0}};
    conda["artifact"]["filename"] = "symphony-qxctl-0.1.0.dev0-fixture_0.conda";
    conda["artifact"]["immutable_uri"] = "https://example.invalid/immutable/conda-1";
    Json auth{{"record_id", "SODV-ART-FIXTURE-AUTH"}, {"record_type", "authorization"}, {"recorded_at", "2026-11-01T00:00:00Z"},
        {"recorded_by", "fixture-recorder"}, {"previous_record_sha256", nullptr},
        {"authority", {{"identity", "fixture-authority"}, {"publisher_identity", "fixture-publisher"}, {"evidence_sha256", sha}}}, {"units", Json::array({unit, conda})}};
    seal(auth);
    Json observations = Json::array();
    for (const auto& item : auth["units"]) observations.push_back({{"unit_id", item["unit_id"]}, {"artifact_sha256", sha}, {"signature_sha256", signature_sha},
        {"provenance_sha256", provenance_sha}, {"inventory_sha256", inventory_sha}, {"download_uri", item["artifact"]["immutable_uri"]}, {"downloaded_bytes", 123},
        {"retrieved_at", "2026-11-01T00:03:00Z"}, {"verification", {{"identity", "fixture-independent-verifier"}, {"signer_identity", "fixture-publisher"}, {"key_id", "SHA256:" + std::string(43, 'A')}, {"allowed_signers_sha256", sha}, {"independent_of_publisher", true},
            {"fresh_download", true}, {"checksum_match", true}, {"signature_verified", true}, {"provenance_verified", true}, {"inventory_verified", true}, {"evidence_sha256", sha}}}});
    Json done{{"record_id", "SODV-ART-FIXTURE-DONE"}, {"record_type", "completion"}, {"recorded_at", "2026-11-01T00:04:00Z"}, {"recorded_by", "fixture-recorder"},
        {"previous_record_sha256", auth["record_sha256"]}, {"authorization_id", auth["record_id"]}, {"authorization_sha256", auth["record_sha256"]},
        {"authorization_merge", {{"source_commit", std::string(40, '3')}, {"merged_at", "2026-11-01T00:01:00Z"}, {"evidence_sha256", sha}}},
        {"published_at", "2026-11-01T00:02:00Z"}, {"observations", observations}};
    seal(done);
    return Json{{"protocol", "symphony.sodv.artifact-release-ledger.v1"}, {"format_version", 1}, {"records", Json::array({auth, done})}};
}
}

void test_sodv_artifacts(const std::filesystem::path& repository) {
    check(check_sodv_artifacts(repository.string()).success, "canonical artifact companion rejected");
    Temporary temporary; const auto path = temporary.path / "knowledge/sodv/ARTIFACT-RELEASES.json";
    const auto valid = fixture(); write(path, valid.dump());
    const auto completed = check_sodv_artifacts(temporary.path.string());
    check(completed.success && completed.records_checked == 2 && completed.completed_transactions == 1, "valid native/conda transaction rejected");
    check(check_sodv_artifacts(fs::relative(temporary.path).string()).success, "relative repository path rejected");
    std::size_t rejected = 0;
    const auto refuse = [&](const std::string& name, const std::function<void(Json&)>& mutation, bool hash = true) {
        auto value = valid; mutation(value); if (hash) reseal(value); write(path, value.dump());
        check(!check_sodv_artifacts(temporary.path.string()).success, "artifact mutation accepted: " + name); ++rejected;
    };
    refuse("unknown field", [](auto& v) { v["records"][0]["unexpected"] = true; });
    refuse("missing authority", [](auto& v) { v["records"][0].erase("authority"); });
    refuse("record digest", [](auto& v) { v["records"][1]["record_sha256"] = std::string(64, 'b'); }, false);
    refuse("record chain", [](auto& v) { v["records"][1]["previous_record_sha256"] = nullptr; seal(v["records"][1]); }, false);
    refuse("duplicate record", [](auto& v) { v["records"][1]["record_id"] = v["records"][0]["record_id"]; });
    refuse("duplicate unit", [](auto& v) { v["records"][0]["units"][1] = v["records"][0]["units"][0]; });
    refuse("duplicate dependency", [](auto& v) { v["records"][0]["units"][0]["dependencies"].push_back(v["records"][0]["units"][0]["dependencies"][0]); });
    refuse("archive is not signed payload", [](auto& v) { v["records"][0]["units"][0]["signature"]["signed_payload_sha256"] = sha; });
    refuse("signature namespace", [](auto& v) { v["records"][0]["units"][0]["signature"]["namespace"] = "unselected"; });
    refuse("conda architecture", [](auto& v) { v["records"][0]["units"][1]["conda"]["subdir"] = "osx-64"; });
    refuse("conda filename", [](auto& v) { v["records"][0]["units"][1]["artifact"]["filename"] = "another.conda"; });
    refuse("native conda data", [](auto& v) { v["records"][0]["units"][0]["conda"] = v["records"][0]["units"][1]["conda"]; });
    refuse("missing completion unit", [](auto& v) { v["records"][1]["observations"].erase(0); });
    refuse("unknown completion unit", [](auto& v) { v["records"][1]["observations"][0]["unit_id"] = "foreign"; });
    refuse("duplicate observation", [](auto& v) { v["records"][1]["observations"][1] = v["records"][1]["observations"][0]; });
    refuse("authorization digest", [](auto& v) { v["records"][1]["authorization_sha256"] = std::string(64, 'b'); });
    refuse("completion before authorization", [](auto& v) { std::swap(v["records"][0], v["records"][1]); });
    refuse("publication before merge", [](auto& v) { v["records"][1]["published_at"] = "2026-11-01T00:00:00Z"; });
    refuse("invalid date", [](auto& v) { v["records"][1]["published_at"] = "2026-02-31T00:00:00Z"; });
    refuse("retrieval before publication", [](auto& v) { v["records"][1]["observations"][0]["retrieved_at"] = "2026-11-01T00:00:00Z"; });
    refuse("publisher verifies itself", [](auto& v) { v["records"][1]["observations"][0]["verification"]["identity"] = "fixture-publisher"; });
    refuse("unselected verification key", [](auto& v) { v["records"][1]["observations"][0]["verification"]["key_id"] = "SHA256:" + std::string(43, 'B'); });
    refuse("unselected verification signer", [](auto& v) { v["records"][1]["observations"][0]["verification"]["signer_identity"] = "foreign"; });
    for (const auto field : {"artifact_sha256", "signature_sha256", "provenance_sha256", "inventory_sha256"})
        refuse(field, [&](auto& v) { v["records"][1]["observations"][0][field] = std::string(64, 'e'); });
    for (const auto field : {"independent_of_publisher", "fresh_download", "checksum_match", "signature_verified", "provenance_verified", "inventory_verified"})
        refuse(field, [&](auto& v) { v["records"][1]["observations"][0]["verification"][field] = false; });
    refuse("download size", [](auto& v) { v["records"][1]["observations"][0]["downloaded_bytes"] = 124; });
    refuse("download locator", [](auto& v) { v["records"][1]["observations"][0]["download_uri"] = "https://example.invalid/other"; });
    refuse("duplicate completion", [](auto& v) { auto second = v["records"][1]; second["record_id"] = "SODV-ART-DUP-DONE"; v["records"].push_back(second); });
    refuse("reused coordinate", [](auto& v) { auto second = v["records"][0]; second["record_id"] = "SODV-ART-SECOND-AUTH"; v["records"] = Json::array({v["records"][0], second}); });
    refuse("conda coordinate ignores relabeling", [](auto& v) {
        auto second = v["records"][0]; second["record_id"] = "SODV-ART-SECOND-AUTH";
        second["units"] = Json::array({second["units"][1]}); auto& u = second["units"][0];
        u["component_id"] = "another-component"; u["unit_id"] = "another-unit";
        u["platform"]["abi"] = "different-abi"; u["conda"]["build_number"] = 1;
        u["artifact"]["immutable_uri"] = "https://example.invalid/other/conda";
        v["records"] = Json::array({v["records"][0], second});
    });
    write(path, "{\"protocol\":\"x\",\"protocol\":\"y\"}"); check(!check_sodv_artifacts(temporary.path.string()).success, "duplicate JSON accepted");
    write(path, std::string(4U * 1024U * 1024U + 1U, ' ')); check(!check_sodv_artifacts(temporary.path.string()).success, "oversized ledger accepted");
    fs::remove(path); check(!check_sodv_artifacts(temporary.path.string()).success, "missing companion accepted");
    const auto target = temporary.path / "target.json"; write(target, valid.dump()); fs::create_symlink(target, path);
    check(!check_sodv_artifacts(temporary.path.string()).success, "symlink companion accepted");
    std::cout << "artifact release companion: native+conda completion, " << rejected << " mutation refusals and four file/parser refusals passed\n";
}
