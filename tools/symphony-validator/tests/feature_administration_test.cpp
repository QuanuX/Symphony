#include "feature_administration.hpp"

#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/json.hpp>

#include <filesystem>
#include <fstream>
#include <iostream>
#include <iterator>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <utility>
#include <unistd.h>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-validator-feature-admin-XXXXXX").string();
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

void require(const bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

bool contains(const FeatureAdministrationCheckResult& result, const std::string& value) {
    for (const auto& message : result.messages) {
        if (message.find(value) != std::string::npos) {
            return true;
        }
    }
    return false;
}

std::string messages(const FeatureAdministrationCheckResult& result) {
    std::string joined;
    for (const auto& message : result.messages) {
        joined += "\n" + message;
    }
    return joined;
}

std::string read_file(const fs::path& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input.good()) {
        throw std::runtime_error("fixture read failed");
    }
    return std::string((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
}

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("fixture write failed");
    }
    output << contents;
}

engine::Json read_json(const fs::path& path) {
    return engine::Json::parse(read_file(path));
}

void write_json(const fs::path& path, const engine::Json& value) {
    write_file(path, value.dump(2) + "\n");
}

void refresh_digest(engine::Json& value, const char* member) {
    value.erase(member);
    value[member] = engine::tagged_sha256(value.dump());
}

void copy_inputs(const fs::path& repository, const fs::path& destination) {
    for (const auto* relative : {
            "knowledge/ssfv/REGISTRY.md",
            "knowledge/FEATURE-ADMINISTRATION-PROFILE.json",
            "tools/qxctl/COMMANDS.json"}) {
        const auto target = destination / relative;
        fs::create_directories(target.parent_path());
        fs::copy_file(repository / relative, target, fs::copy_options::overwrite_existing);
    }
}

std::size_t registry_feature_count(const fs::path& repository) {
    constexpr std::string_view prefix = "- feature_id: `";
    std::istringstream input(read_file(repository / "knowledge/ssfv/REGISTRY.md"));
    std::string line;
    std::size_t count = 0U;
    while (std::getline(input, line)) {
        if (line.starts_with(prefix)) {
            ++count;
        }
    }
    return count;
}

std::size_t expected_unreviewed_count(const fs::path& repository) {
    const auto profile = read_json(repository / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json");
    std::size_t count = 0U;
    for (const auto& feature : profile.at("features")) {
        if (feature.at("expectations").empty()) {
            ++count;
            continue;
        }
        for (const auto& expectation : feature.at("expectations")) {
            if (expectation.at("delivery") == "unreviewed") {
                ++count;
            }
        }
    }
    return count;
}

void test_canonical(const fs::path& repository) {
    const auto expected_features = registry_feature_count(repository);
    const auto expected_unreviewed = expected_unreviewed_count(repository);
    const auto result = check_feature_administration(repository.string());
    require(result.success, "canonical feature-administration assurance failed:" + messages(result));
    require(result.features_checked == expected_features, "canonical feature count mismatch");
    require(result.commands_checked > 0U, "canonical command inventory is empty");
    require(result.unreviewed_features == expected_unreviewed,
        "canonical bootstrap debt count mismatch");
    require(contains(result,
        "feature_administration.scan_complete features=" + std::to_string(expected_features) +
        " commands="),
        "canonical completion evidence missing");
}

void test_profile_bindings(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto registry = read_file(temporary.path() / "knowledge/ssfv/REGISTRY.md");
        registry += "\n- feature_id: `ssfv:zzzz:future-feature`\n";
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md", registry);
        auto profile = read_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json");
        profile["features"].push_back(engine::Json{
            {"expectations", engine::Json::array()},
            {"feature_id", "ssfv:zzzz:future-feature"},
        });
        profile["registered_feature_count"] = profile["features"].size();
        profile["ssfv_registry_digest"] = engine::tagged_sha256(registry);
        refresh_digest(profile, "profile_digest");
        write_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", profile);
        const auto result = check_feature_administration(temporary.path().string());
        require(result.success,
            "a digest-bound future registry/profile feature was rejected:" + messages(result));
        require(result.features_checked == registry_feature_count(repository) + 1U,
            "future feature count was not derived from the registry");
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto registry = read_file(temporary.path() / "knowledge/ssfv/REGISTRY.md");
        registry.append("\n");
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md", registry);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.profile_registry_digest"),
            "SSFV registry byte drift was accepted");
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto profile = read_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json");
        profile["profile_id"] = "tampered.profile";
        write_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", profile);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.profile_digest"),
            "invalid profile self-digest was accepted");
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto profile = read_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json");
        profile["features"].erase(profile["features"].end() - 1);
        profile["registered_feature_count"] = profile["features"].size();
        refresh_digest(profile, "profile_digest");
        write_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", profile);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.profile_feature_set"),
            "incomplete profile feature set was accepted");
    }
}

void test_command_registry(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto commands = read_json(temporary.path() / "tools/qxctl/COMMANDS.json");
        commands["client_id"] = "tampered";
        write_json(temporary.path() / "tools/qxctl/COMMANDS.json", commands);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.commands_shape"),
            "invalid expected command registry identity was accepted");
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto commands = read_json(temporary.path() / "tools/qxctl/COMMANDS.json");
        std::swap(commands["commands"][0], commands["commands"][1]);
        refresh_digest(commands, "registry_digest");
        write_json(temporary.path() / "tools/qxctl/COMMANDS.json", commands);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.command_order"),
            "unsorted command IDs were accepted");
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto commands = read_json(temporary.path() / "tools/qxctl/COMMANDS.json");
        commands["commands"][1]["command_id"] = commands["commands"][0]["command_id"];
        refresh_digest(commands, "registry_digest");
        write_json(temporary.path() / "tools/qxctl/COMMANDS.json", commands);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.command_id"),
            "duplicate command ID was accepted");
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto commands = read_json(temporary.path() / "tools/qxctl/COMMANDS.json");
        commands["commands"][0]["feature_bindings"][0]["feature_id"] = "ssfv:unknown:missing";
        refresh_digest(commands, "registry_digest");
        write_json(temporary.path() / "tools/qxctl/COMMANDS.json", commands);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result, "feature_administration.command_feature_binding"),
            "unregistered command feature binding was accepted");
    }
}

void test_forward_gate_and_command_reference(const fs::path& repository) {
    for (const auto* gate : {"enforce_new_records", "enforce_all_records"}) {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto profile = read_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json");
        profile["forward_gate"] = gate;
        refresh_digest(profile, "profile_digest");
        write_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", profile);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result,
            "feature_administration.profile_unreviewed feature_id="),
            std::string("empty expectations passed ") + gate);
    }
    {
        TemporaryDirectory temporary;
        copy_inputs(repository, temporary.path());
        auto profile = read_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json");
        profile["features"][0]["expectations"] = engine::Json::array({engine::Json{
            {"command_ids", engine::Json::array({"qxcmd:symphony:missing"})},
            {"delivery", "direct"},
            {"engine_operation_ids", engine::Json::array()},
            {"evidence", engine::Json::array({"fixture"})},
            {"inherited_from_feature_id", nullptr},
            {"interaction", "discover"},
            {"rationale", "Fixture exercises reference resolution."},
            {"requirement", "required"},
        }});
        refresh_digest(profile, "profile_digest");
        write_json(temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", profile);
        const auto result = check_feature_administration(temporary.path().string());
        require(!result.success && contains(result,
            "feature_administration.profile_command_reference command_id=qxcmd:symphony:missing"),
            "unresolved profile command reference was accepted");
    }
}

void test_no_follow(const fs::path& repository) {
    TemporaryDirectory temporary;
    copy_inputs(repository, temporary.path());
    const auto profile = temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json";
    const auto target = temporary.path() / "profile-target.json";
    fs::rename(profile, target);
    fs::create_symlink(target, profile);
    const auto result = check_feature_administration(temporary.path().string());
    require(!result.success && contains(result,
        "feature_administration.unreadable path=knowledge/FEATURE-ADMINISTRATION-PROFILE.json"),
        "symlinked profile was accepted");
}

void test_surface_closure(const fs::path& repository) {
    TemporaryDirectory temporary;
    const auto profile = temporary.path() / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json";
    fs::create_directories(profile.parent_path());
    fs::copy_file(repository / "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", profile);
    const auto result = check_feature_administration(temporary.path().string());
    require(!result.success &&
        contains(result, "feature_administration.unreadable path=knowledge/ssfv/REGISTRY.md code=missing") &&
        contains(result, "feature_administration.unreadable path=tools/qxctl/COMMANDS.json code=missing"),
        "partial feature-administration surface was accepted");
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root required");
        }
        const auto repository = fs::canonical(argv[1]);
        test_canonical(repository);
        test_profile_bindings(repository);
        test_command_registry(repository);
        test_forward_gate_and_command_reference(repository);
        test_no_follow(repository);
        test_surface_closure(repository);
        std::cout << "feature administration validator tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "feature administration validator tests failed: " << error.what() << '\n';
        return 1;
    }
}
