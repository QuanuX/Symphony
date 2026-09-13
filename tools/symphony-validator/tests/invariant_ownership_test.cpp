#include "invariant_ownership.hpp"

#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/json.hpp>

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <iterator>
#include <set>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-invariant-ownership-XXXXXX").string();
        pattern.push_back('\0');
        char* value = ::mkdtemp(pattern.data());
        if (value == nullptr) {
            throw std::runtime_error("mkdtemp failed");
        }
        path_ = value;
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

void require(const bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

std::string messages(const InvariantOwnershipCheckResult& result) {
    std::string output;
    for (const auto& message : result.messages) {
        output += "\n" + message;
    }
    return output;
}

bool contains(const InvariantOwnershipCheckResult& result, const std::string& text) {
    for (const auto& message : result.messages) {
        if (message.find(text) != std::string::npos) {
            return true;
        }
    }
    return false;
}

std::string read_file(const fs::path& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input.good()) {
        throw std::runtime_error("read failed: " + path.string());
    }
    return std::string((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
}

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("write failed: " + path.string());
    }
    output << contents;
}

engine::Json read_json(const fs::path& path) {
    return engine::Json::parse(read_file(path));
}

void refresh_digest(engine::Json& registry) {
    registry.erase("registry_digest");
    registry["registry_digest"] = engine::tagged_sha256(registry.dump());
}

void write_registry(const fs::path& destination, engine::Json registry) {
    refresh_digest(registry);
    write_file(destination / "knowledge/INVARIANT-OWNERSHIP.json", registry.dump(2) + "\n");
}

void copy_regular(const fs::path& repository, const fs::path& destination, const std::string& relative) {
    const auto target = destination / relative;
    fs::create_directories(target.parent_path());
    fs::copy_file(repository / relative, target, fs::copy_options::overwrite_existing);
}

void copy_fixture(const fs::path& repository, const fs::path& destination) {
    const auto registry = read_json(repository / "knowledge/INVARIANT-OWNERSHIP.json");
    copy_regular(repository, destination, "knowledge/INVARIANT-OWNERSHIP.json");
    for (const auto& adapter : registry.at("adapters")) {
        copy_regular(repository, destination, adapter.at("owner_contract").get<std::string>());
        const auto implementation_path = adapter.at("implementation_path").get<std::string>();
        fs::create_directories(destination / implementation_path);
        if (adapter.at("adapter_id") == "adapter:symphony:ssiag.macos-keychain-provider.v1") {
            copy_regular(repository, destination, implementation_path + "/Protocol.swift");
            copy_regular(repository, destination, implementation_path + "/SignedBundleReadiness.swift");
        }
    }
    for (const auto& invariant : registry.at("invariants")) {
        copy_regular(repository, destination, invariant.at("owner_contract").get<std::string>());
        for (const auto& path : invariant.at("producer_implementations")) {
            copy_regular(repository, destination, path.get<std::string>());
        }
        for (const auto* family : {"producer_regressions", "consumer_boundary_rejections",
                "real_process_regressions"}) {
            for (const auto& reference : invariant.at(family)) {
                copy_regular(repository, destination, reference.at("path").get<std::string>());
            }
        }
    }
}

void test_absent() {
    TemporaryDirectory temporary;
    const auto result = check_invariant_ownership(temporary.path().string());
    require(result.success && contains(result, "invariant_ownership.absent registry=false"),
        "legacy absence did not remain explicitly compatible");
}

engine::Json generic_registry_fixture(const fs::path& destination) {
    write_file(destination / "modules/example-engine/SPEC.md", "# Exact example owner\n");
    write_file(destination / "modules/example-engine/src/main.cpp", "int main() { return 0; }\n");
    write_file(destination / "modules/example-engine/tests/owner.cpp", "void owner_rejects_bad_state() {}\n");
    write_file(destination / "modules/example-engine/tests/consumer.cpp", "void consumer_rejects_bad_result() {}\n");
    write_file(destination / "modules/example-engine/tests/process.py",
        "def installed_process():\n"
        "    receipt = 'install-receipt.json'\n"
        "    result = subprocess.run([receipt], input=b'{}', capture_output=True)\n"
        "    assert result.returncode == 0\n");
    return engine::Json{
        {"protocol", "symphony.knowledge.invariant-ownership-registry.v2"},
        {"format_version", 2U}, {"scope", "common_lowest_authoritative_layer"},
        {"catalog_scope", "registered_incremental"}, {"catalog_complete", false},
        {"forward_gate", "enforce_new_or_modified"},
        {"test_policy", {{"consumer_boundary_rejection_required", true},
            {"owner_producer_regression_required", true}, {"real_process_required_for_ipc", true}}},
        {"adapters", engine::Json::array({{
            {"adapter_id", "adapter:symphony:symphony-example.v1"},
            {"command_protocol", "symphony.knowledge.engine-process.v1"},
            {"component", "example-engine"}, {"entry_point_id", "symphony-example"},
            {"format_version", 2U}, {"implementation_path", "modules/example-engine"},
            {"operation_ids", engine::Json::array({"engop:symphony:example.query"})},
            {"owner_contract", "modules/example-engine/SPEC.md"},
            {"version_policy", "exact_receipt_v2_entry_point_and_capability_compatible"}
        }})},
        {"invariants", engine::Json::array({{
            {"invariant_id", "invariant:symphony:example.result-lineage"},
            {"title", "Example result lineage"}, {"owner_contract", "modules/example-engine/SPEC.md"},
            {"owner_component", "example-engine"}, {"statement", "Retain exact supplied lineage."},
            {"producer_implementations", engine::Json::array({"modules/example-engine/src/main.cpp"})},
            {"producer_regressions", engine::Json::array({{{"path", "modules/example-engine/tests/owner.cpp"},
                {"cases", engine::Json::array({"owner_rejects_bad_state"})}}})},
            {"consumer_boundary_rejections", engine::Json::array({{{"path", "modules/example-engine/tests/consumer.cpp"},
                {"cases", engine::Json::array({"consumer_rejects_bad_result"})}}})},
            {"allowed_adapter_ids", engine::Json::array({"adapter:symphony:symphony-example.v1"})},
            {"ipc_boundary", true},
            {"real_process_regressions", engine::Json::array({{{"path", "modules/example-engine/tests/process.py"},
                {"cases", engine::Json::array({"installed_process"})}}})},
            {"status", "active"}
        }})}
    };
}

void test_v3_explicit_adapter_ownership() {
    TemporaryDirectory temporary;
    auto fixture = generic_registry_fixture(temporary.path());
    fs::rename(temporary.path() / "modules/example-engine", temporary.path() / "modules/storage-adapter");
    auto encoded = fixture.dump();
    const std::string old_name = "example-engine", new_name = "storage-adapter";
    std::size_t position = 0;
    while ((position = encoded.find(old_name, position)) != std::string::npos) {
        encoded.replace(position, old_name.size(), new_name); position += new_name.size();
    }
    fixture = engine::Json::parse(encoded);
    fixture["protocol"] = "symphony.knowledge.invariant-ownership-registry.v3";
    fixture["format_version"] = 3U;
    fixture["adapters"][0]["format_version"] = 3U;
    fixture["adapters"][0]["operation_ids"] = engine::Json::array({"engop:symphony:scv.graph-index.query"});
    write_registry(temporary.path(), fixture);
    auto result = check_invariant_ownership(temporary.path().string());
    require(result.success, "explicit v3 adapter rejected independent module/operation names:" + messages(result));
    for (const auto old_version : {1U, 2U}) {
        auto wrong = fixture;
        wrong["protocol"] = "symphony.knowledge.invariant-ownership-registry.v" + std::to_string(old_version);
        wrong["format_version"] = old_version;
        write_registry(temporary.path(), wrong);
        result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.adapter_shape"), "explicit adapter widened an older registry");
    }
    auto wrong = fixture; wrong["adapters"][0]["format_version"] = 2U;
    write_registry(temporary.path(), wrong); result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "invariant_ownership.adapter_identity"), "v3 relaxed existing v2 adapter identity");
    wrong = fixture; wrong["adapters"][0]["implementation_path"] = "modules/storage-adapter/src";
    write_registry(temporary.path(), wrong); result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "explicit_implementation_mismatch"), "explicit adapter accepted a substituted implementation owner");
    wrong = fixture; wrong["adapters"][0]["owner_contract"] = "modules/example-engine/SPEC.md";
    write_registry(temporary.path(), wrong); result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "generic_owner_mismatch"), "explicit adapter accepted a substituted contract owner");
    wrong = fixture; wrong["adapters"][0]["adapter_id"] = "adapter:symphony:symphony-other.v1";
    write_registry(temporary.path(), wrong); result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "invariant_ownership.adapter_identity"), "explicit adapter accepted a mismatched entrypoint identity");
}

void test_v2_generic_adapter() {
    TemporaryDirectory temporary;
    const auto fixture = generic_registry_fixture(temporary.path());
    write_registry(temporary.path(), fixture);
    auto result = check_invariant_ownership(temporary.path().string());
    require(result.success, "generic v2 adapter traceability failed:" + messages(result));
    auto wrong = fixture;
    wrong["protocol"] = "symphony.knowledge.invariant-ownership-registry.v1";
    wrong["format_version"] = 1U;
    write_registry(temporary.path(), wrong);
    result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "invariant_ownership.adapter_shape"),
        "generic engine adapter widened the unchanged v1 protocol");
    wrong = fixture;
    wrong["adapters"][0]["entry_point_id"] = "symphony-other";
    write_registry(temporary.path(), wrong);
    result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "invariant_ownership.adapter_identity"),
        "generic adapter accepted unrelated receipt entrypoint");
    wrong = fixture;
    wrong["adapters"][0]["operation_ids"] = engine::Json::array({"engop:symphony:other.query"});
    write_registry(temporary.path(), wrong);
    result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "reason=generic_domain_mismatch"),
        "generic adapter claimed a different domain operation");
    write_registry(temporary.path(), fixture);
    write_file(temporary.path() / "modules/example-engine/tests/process.py",
        "def installed_process():\n    pass\n");
    result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "invariant_ownership.real_process_mechanics"),
        "Python placeholder passed installed process traceability");
}

void test_canonical(const fs::path& repository) {
    const auto registry = read_json(repository / "knowledge/INVARIANT-OWNERSHIP.json");
    const auto expected_invariants = registry.at("invariants").size();
    const auto expected_adapters = registry.at("adapters").size();
    std::size_t expected_evidence = 0U;
    for (const auto& invariant : registry.at("invariants")) {
        expected_evidence += invariant.at("producer_regressions").size();
        expected_evidence += invariant.at("consumer_boundary_rejections").size();
        expected_evidence += invariant.at("real_process_regressions").size();
    }
    require(expected_invariants > 0U, "canonical invariant inventory is empty");
    const auto result = check_invariant_ownership(repository.string());
    require(result.success, "canonical invariant ownership failed:" + messages(result));
    require(result.invariants_checked == expected_invariants, "canonical invariant count mismatch");
    require(result.adapters_checked == expected_adapters, "canonical adapter count mismatch");
    require(result.evidence_references_checked == expected_evidence, "canonical evidence count mismatch");
    require(contains(result,
        "invariant_ownership.scan_complete invariants=" + std::to_string(expected_invariants) +
        " adapters=" + std::to_string(expected_adapters) +
        " evidence_references=" + std::to_string(expected_evidence) + " violations=0"),
        "canonical completion evidence missing");
}

void test_v1_registry_compatibility(const fs::path& repository) {
    TemporaryDirectory temporary;
    copy_fixture(repository, temporary.path());
    auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
    std::set<std::string> removed;
    auto retained_adapters = engine::Json::array();
    for (const auto& adapter : registry.at("adapters")) {
        if (adapter.at("format_version") == 1U) retained_adapters.push_back(adapter);
        else removed.insert(adapter.at("adapter_id").get<std::string>());
    }
    auto retained_invariants = engine::Json::array();
    for (const auto& invariant : registry.at("invariants")) {
        const bool needs_generic = std::any_of(invariant.at("allowed_adapter_ids").begin(),
            invariant.at("allowed_adapter_ids").end(), [&removed](const engine::Json& id) {
                return removed.contains(id.get<std::string>());
            });
        if (!needs_generic) retained_invariants.push_back(invariant);
    }
    registry["adapters"] = retained_adapters;
    registry["invariants"] = retained_invariants;
    registry["protocol"] = "symphony.knowledge.invariant-ownership-registry.v1";
    registry["format_version"] = 1U;
    write_registry(temporary.path(), registry);
    auto result = check_invariant_ownership(temporary.path().string());
    require(result.success, "unchanged v1 registry was rejected:" + messages(result));
    registry["protocol"] = "symphony.knowledge.invariant-ownership-registry.v2";
    write_registry(temporary.path(), registry);
    result = check_invariant_ownership(temporary.path().string());
    require(!result.success && contains(result, "invariant_ownership.registry_shape"),
        "mismatched registry protocol and format passed");
}

void test_shape_digest_and_order(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["scope"] = "tampered";
        write_file(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json", registry.dump());
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.registry_shape"),
            "wrong fixed registry value passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["catalog_complete"] = false;
        registry["registry_digest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000";
        write_file(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json", registry.dump());
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.registry_digest expected="),
            "stale recursive omit-self digest passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        std::swap(registry["invariants"][0], registry["invariants"][1]);
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.invariant_order"),
            "unsorted invariants passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        std::swap(registry["adapters"][0], registry["adapters"][1]);
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.adapter_order"),
            "unsorted adapters passed");
    }
}

void test_adapter_closure(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        auto& references = registry["invariants"][0]["allowed_adapter_ids"];
        references.push_back("adapter:symphony:unknown.v1");
        std::sort(references.begin(), references.end());
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.adapter_reference_unresolved"),
            "unresolved adapter reference passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        auto operation = registry["adapters"][0]["operation_ids"][0];
        registry["adapters"][1]["operation_ids"].push_back(operation);
        std::sort(registry["adapters"][1]["operation_ids"].begin(),
            registry["adapters"][1]["operation_ids"].end());
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.adapter_operation_owner"),
            "operation assigned to multiple adapters passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        auto provider = std::find_if(registry["adapters"].begin(), registry["adapters"].end(),
            [](const engine::Json& adapter) {
                return adapter.at("adapter_id") == "adapter:symphony:ssiag.macos-keychain-provider.v1";
            });
        require(provider != registry["adapters"].end(), "canonical provider adapter missing");
        (*provider)["operation_ids"] = engine::Json::array({
            "engop:symphony:ssiag.provider.metadata-invented",
        });
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "reason=not_adapter_owned_operation_set"),
            "invented macOS provider operation passed");
    }
}

void test_identifier_grammar(const fs::path& repository) {
    for (const auto& invalid : {"invariant:symphony:a..b", "invariant:symphony:trailing-"}) {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["invariants"][0]["invariant_id"] = invalid;
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.invariant_shape"),
            std::string("malformed invariant ID passed: ") + invalid);
    }
    for (const auto& invalid : {"adapter:symphony:a..b", "adapter:symphony:trailing-"}) {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["adapters"][0]["adapter_id"] = invalid;
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.adapter_shape"),
            std::string("malformed adapter ID passed: ") + invalid);
    }
    for (const auto& invalid : {"engop::missing.namespace", "engop:Symphony:bad",
            "engop:symphony:", "engop:symphony:a..b"}) {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["adapters"][0]["operation_ids"][0] = invalid;
        std::sort(registry["adapters"][0]["operation_ids"].begin(),
            registry["adapters"][0]["operation_ids"].end());
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.adapter_operations"),
            std::string("malformed operation ID passed: ") + invalid);
    }
}

void test_regression_evidence(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["invariants"][0]["producer_regressions"][0]["cases"].push_back(
            registry["invariants"][0]["producer_regressions"][0]["cases"][0]);
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.regression_cases"),
            "duplicate test case passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["invariants"][0]["producer_regressions"][0]["cases"][0] = "TestMissingCase";
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.regression_case_missing"),
            "missing named test case passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["invariants"][0]["producer_regressions"][0]["path"] = "tests/missing_test.go";
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result,
            "invariant_ownership.reference_unreadable path=tests/missing_test.go"),
            "missing evidence path passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        const auto& reference = registry["invariants"][0]["producer_regressions"][0];
        registry["invariants"][0]["consumer_boundary_rejections"] =
            engine::Json::array({reference});
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.regression_role_collision"),
            "one test case fulfilled two evidence roles");
    }
}

void test_ipc_policy(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        registry["invariants"][1]["real_process_regressions"].erase(
            registry["invariants"][1]["real_process_regressions"].end() - 1);
        write_registry(temporary.path(), registry);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result,
            "invariant_ownership.real_process_adapter_coverage"),
            "IPC invariant without per-adapter real-process coverage passed");
    }
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        const auto path = registry["invariants"][1]["real_process_regressions"][0]["path"].get<std::string>();
        write_file(temporary.path() / path,
            "package main\nimport \"testing\"\nfunc TestFoundationLifecycleRealProcessBoundary(t *testing.T) {}\n");
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.real_process_mechanics"),
            "source-only placeholder satisfied real-process evidence");
    }
}

void test_no_follow_and_bounds(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_fixture(repository, temporary.path());
        auto registry = read_json(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json");
        const auto path = registry["invariants"][0]["producer_regressions"][0]["path"].get<std::string>();
        const auto fixture = temporary.path() / path;
        const auto target = temporary.path() / "outside_test.go";
        fs::rename(fixture, target);
        fs::create_symlink(target, fixture);
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result,
            "invariant_ownership.reference_unreadable path=" + path),
            "symlinked evidence path passed");
    }
    {
        TemporaryDirectory temporary;
        fs::create_directories(temporary.path() / "knowledge");
        write_file(temporary.path() / "knowledge/INVARIANT-OWNERSHIP.json",
            std::string(4U * 1024U * 1024U + 1U, ' '));
        const auto result = check_invariant_ownership(temporary.path().string());
        require(!result.success && contains(result, "invariant_ownership.unreadable"),
            "oversized registry passed");
    }
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root required");
        }
        const auto repository = fs::canonical(argv[1]);
        test_absent();
        test_v2_generic_adapter();
        test_v3_explicit_adapter_ownership();
        test_canonical(repository);
        test_v1_registry_compatibility(repository);
        test_shape_digest_and_order(repository);
        test_adapter_closure(repository);
        test_identifier_grammar(repository);
        test_regression_evidence(repository);
        test_ipc_policy(repository);
        test_no_follow_and_bounds(repository);
        std::cout << "invariant ownership tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "invariant ownership tests failed: " << error.what() << "\n";
        return 1;
    }
}
