#include "lifecycle.hpp"

#include "coordinator.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <cctype>
#include <cstdint>
#include <functional>
#include <iterator>
#include <map>
#include <optional>
#include <set>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::session {
namespace engine = symphony::knowledge::engine;

namespace {

constexpr std::size_t max_components = 4096U;
constexpr std::size_t max_dependencies = 256U;
constexpr std::size_t max_packages = 256U;
constexpr std::size_t max_extensions = 64U;
constexpr std::size_t max_capabilities = 128U;
constexpr std::size_t max_actions = 4096U;

const std::vector<std::string> base_required_capabilities = {
    "dependency-ready-set-v1",
    "deterministic-action-id-v1",
    "forward-inverse-v1",
    "localized-blocker-isolation-v1",
    "ordered-safety-phases-v1",
    "report-only-v1",
    "unknown-critical-block-v1",
};

const std::vector<std::string> supported_capabilities = {
    "dependency-ready-set-v1",
    "deterministic-action-id-v1",
    "forward-inverse-v1",
    "localized-blocker-isolation-v1",
    "ordered-safety-phases-v1",
    "receipt-v1-adapter",
    "receipt-v2",
    "report-only-v1",
    "unknown-critical-block-v1",
};

struct Dependency final {
    std::string target;
    std::string condition;
    bool critical{false};
};

struct DesiredComponent final {
    std::string id;
    std::string kind;
    std::string module_id;
    std::optional<std::string> vector_id;
    std::optional<std::string> engine_id;
    std::string presence;
    std::optional<std::string> selected_receipt;
    std::optional<int> selected_receipt_version;
    bool required{false};
    std::string activation;
    std::string docking;
    std::optional<std::string> receptor_id;
    std::string target_state_digest;
    std::set<std::string> required_capabilities;
    std::vector<Dependency> dependencies;
};

struct ObservedPackage final {
    std::string receipt_protocol;
    std::string receipt_digest;
    std::string integrity;
    bool entry_points_validated{false};
};

struct ObservedComponent final {
    std::string id;
    std::string kind;
    std::string module_id;
    std::optional<std::string> vector_id;
    std::optional<std::string> engine_id;
    std::vector<ObservedPackage> packages;
    std::optional<std::string> selected_receipt;
    std::string activation;
    std::string docking;
    std::optional<std::string> receptor_id;
    std::set<std::string> capabilities;
    std::string platform_compatibility;
    std::string observation_digest;
};

struct ParsedDesired final {
    std::string profile_id;
    std::string tops_id;
    std::string digest;
    std::map<std::string, DesiredComponent> components;
    std::set<int> receipt_versions;
    bool critical_extension{false};
};

struct ParsedObservation final {
    std::string profile_id;
    std::string tops_id;
    std::string digest;
    std::string stable_inventory_digest;
    std::optional<std::string> binding_registry_digest;
    std::string platform_digest;
    std::map<std::string, ObservedComponent> components;
    std::set<int> receipt_versions;
    bool critical_unknown{false};
};

struct DraftAction final {
    std::string component_id;
    std::string kind;
    std::string inverse_kind;
    std::string direction;
    std::optional<std::string> expected_before;
    std::string target_state_digest;
    std::optional<std::string> target_receptor_id;
    std::vector<std::string> artifact_digests;
    std::vector<std::string> expected_evidence;
    std::vector<std::size_t> prerequisites;
    std::vector<engine::Json> blockers;
    std::string semantic_key;
    std::string action_id;
    std::optional<std::string> inverse_action_id;
};

[[noreturn]] void invalid(std::string code, std::string detail) {
    throw engine::Error(std::move(code), std::move(detail), 4);
}

void check_deadline(std::int64_t deadline_unix_ms) {
    if (engine::unix_time_ms() >= deadline_unix_ms) {
        invalid("request.deadline_expired", "lifecycle planning deadline expired");
    }
}

void exact_fields(
    const engine::Json& value,
    const std::set<std::string>& expected,
    std::string_view context) {
    if (!value.is_object() || value.size() != expected.size()) {
        invalid("lifecycle.field_set", std::string(context) + " has an invalid field set");
    }
    for (const auto& [key, item] : value.items()) {
        static_cast<void>(item);
        if (!expected.contains(key)) {
            invalid("lifecycle.unknown_field", std::string(context) + " contains an unknown field");
        }
    }
}

bool is_token(std::string_view text, std::size_t maximum = 256U) {
    if (text.empty() || text.size() > maximum) return false;
    return std::all_of(text.begin(), text.end(), [](const unsigned char character) {
        return std::isalnum(character) != 0 || character == '.' || character == '_' ||
               character == ':' || character == '-';
    });
}

bool is_digest(std::string_view text) {
    return text.size() == 71U && text.starts_with("sha256:") &&
        std::all_of(text.begin() + 7, text.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') ||
                   (character >= 'a' && character <= 'f');
        });
}

bool is_version(std::string_view text) {
    if (text.empty() || text.size() > 64U) return false;
    return std::all_of(text.begin(), text.end(), [](const unsigned char character) {
        return std::isalnum(character) != 0 || character == '.' || character == '+' || character == '-';
    });
}

const std::string& text_field(
    const engine::Json& object,
    const char* field,
    std::size_t maximum = 256U) {
    const auto& value = object.at(field);
    if (!value.is_string()) invalid("lifecycle.invalid_field", std::string(field) + " must be a string");
    const auto& text = value.get_ref<const std::string&>();
    if (!is_token(text, maximum)) invalid("lifecycle.invalid_field", std::string(field) + " has invalid syntax");
    return text;
}

const std::string& version_field(const engine::Json& object, const char* field) {
    const auto& value = object.at(field);
    if (!value.is_string() || !is_version(value.get_ref<const std::string&>())) {
        invalid("lifecycle.invalid_field", std::string(field) + " has invalid version syntax");
    }
    return value.get_ref<const std::string&>();
}

std::string digest_field(const engine::Json& object, const char* field) {
    const auto& value = object.at(field);
    if (!value.is_string() || !is_digest(value.get_ref<const std::string&>())) {
        invalid("lifecycle.invalid_digest", std::string(field) + " has invalid digest syntax");
    }
    return value.get<std::string>();
}

std::optional<std::string> optional_token(const engine::Json& value, const char* field) {
    if (value.is_null()) return std::nullopt;
    if (!value.is_string() || !is_token(value.get_ref<const std::string&>())) {
        invalid("lifecycle.invalid_field", std::string(field) + " must be a token or null");
    }
    return value.get<std::string>();
}

std::optional<std::string> optional_digest(const engine::Json& value, const char* field) {
    if (value.is_null()) return std::nullopt;
    if (!value.is_string() || !is_digest(value.get_ref<const std::string&>())) {
        invalid("lifecycle.invalid_digest", std::string(field) + " must be a digest or null");
    }
    return value.get<std::string>();
}

bool boolean_field(const engine::Json& object, const char* field) {
    if (!object.at(field).is_boolean()) {
        invalid("lifecycle.invalid_field", std::string(field) + " must be a boolean");
    }
    return object.at(field).get<bool>();
}

void require_const(const engine::Json& value, const engine::Json& expected, const char* field) {
    if (value != expected) invalid("lifecycle.invalid_field", std::string(field) + " is unsupported");
}

void require_enum(const std::string& value, const std::set<std::string>& allowed, const char* field) {
    if (!allowed.contains(value)) invalid("lifecycle.invalid_field", std::string(field) + " is unsupported");
}

bool safe_absolute_path(std::string_view path) {
    if (path.empty() || path.size() > 4096U || path.front() != '/' || path.find('\\') != std::string_view::npos) {
        return false;
    }
    if (path.size() > 1U && path.back() == '/') return false;
    std::size_t start = 1U;
    while (start <= path.size()) {
        const auto end = path.find('/', start);
        const auto part = path.substr(start, end == std::string_view::npos ? path.size() - start : end - start);
        if (part.empty() || part == "." || part == "..") return false;
        if (std::any_of(part.begin(), part.end(), [](const unsigned char character) {
            return character < 0x20U || character == 0x7fU;
        })) return false;
        if (end == std::string_view::npos) break;
        start = end + 1U;
    }
    return true;
}

void sort_array(engine::Json& array) {
    std::sort(array.begin(), array.end(), [](const engine::Json& left, const engine::Json& right) {
        return left.dump() < right.dump();
    });
}

void normalize_extensions(engine::Json& extensions) {
    for (auto& extension : extensions) {
        if (extension.is_object() && extension.contains("payload")) {
            static_cast<void>(extension.at("payload"));
        }
    }
    sort_array(extensions);
}

engine::Json normalize_desired(engine::Json value) {
    for (auto& component : value.at("components")) {
        sort_array(component.at("dependencies"));
        sort_array(component.at("compatibility").at("required_capabilities"));
        sort_array(component.at("compatibility").at("platform_requirements"));
        normalize_extensions(component.at("extensions"));
    }
    sort_array(value.at("components"));
    normalize_extensions(value.at("extensions"));
    return value;
}

engine::Json normalize_observation(engine::Json value) {
    sort_array(value.at("configured_roots"));
    sort_array(value.at("platform").at("provider_availability"));
    for (auto& component : value.at("components")) {
        sort_array(component.at("packages"));
        sort_array(component.at("capabilities"));
    }
    sort_array(value.at("components"));
    sort_array(value.at("unknown_packages"));
    return value;
}

std::string verify_document_digest(
    engine::Json value,
    const char* field,
    const std::function<engine::Json(engine::Json)>& normalize) {
    const auto expected = digest_field(value, field);
    value.erase(field);
    value = normalize(std::move(value));
    const auto actual = engine::tagged_sha256(value.dump());
    if (actual != expected) invalid("lifecycle.digest_mismatch", std::string(field) + " does not match normalized content");
    return expected;
}

bool validate_extensions(const engine::Json& extensions) {
    if (!extensions.is_array() || extensions.size() > max_extensions) {
        invalid("lifecycle.invalid_extensions", "extensions must be a bounded array");
    }
    std::set<std::string> identities;
    bool critical = false;
    for (const auto& extension : extensions) {
        exact_fields(extension, {
            "extension_id", "extension_version", "critical", "payload", "payload_digest"
        }, "extension");
        const auto id = text_field(extension, "extension_id");
        const auto version = version_field(extension, "extension_version");
        if (!identities.insert(id + "@" + version).second) {
            invalid("lifecycle.duplicate_extension", "extension identity is duplicated");
        }
        const auto payload_digest = digest_field(extension, "payload_digest");
        if (payload_digest != engine::tagged_sha256(extension.at("payload").dump())) {
            invalid("lifecycle.digest_mismatch", "extension payload digest does not match");
        }
        critical = boolean_field(extension, "critical") || critical;
    }
    return critical;
}

int receipt_version(std::string_view protocol);

ParsedDesired parse_desired(const engine::Json& value, std::int64_t deadline_unix_ms) {
    exact_fields(value, {
        "protocol", "format_version", "profile_id", "tops_id", "generation",
        "previous_desired_state_digest", "components", "extensions", "canonical",
        "desired_state_digest"
    }, "desired state");
    require_const(value.at("protocol"), "symphony.knowledge.lifecycle-desired-state.v1", "desired protocol");
    require_const(value.at("format_version"), 1, "desired format version");
    require_const(value.at("canonical"), false, "desired canonical status");
    if (!value.at("generation").is_number_integer() || value.at("generation").get<std::int64_t>() < 1) {
        invalid("lifecycle.invalid_field", "desired generation is invalid");
    }
    static_cast<void>(optional_digest(value.at("previous_desired_state_digest"), "previous_desired_state_digest"));
    ParsedDesired parsed{
        text_field(value, "profile_id"),
        text_field(value, "tops_id"),
        digest_field(value, "desired_state_digest"),
        {}, {},
        validate_extensions(value.at("extensions")),
    };
    if (!value.at("components").is_array() || value.at("components").size() > max_components) {
        invalid("lifecycle.invalid_components", "desired components must be a bounded array");
    }
    for (const auto& component : value.at("components")) {
        check_deadline(deadline_unix_ms);
        exact_fields(component, {
            "component_id", "component_kind", "module_id", "vector_id", "engine_id",
            "presence", "selected_package", "required", "install_scope", "install_root",
            "activation", "docking", "dependencies", "compatibility", "extensions"
        }, "desired component");
        DesiredComponent output;
        output.id = text_field(component, "component_id");
        output.kind = text_field(component, "component_kind");
        require_enum(output.kind, {"coordinator", "vector_engine", "module", "adapter", "ui", "service"}, "component_kind");
        output.module_id = text_field(component, "module_id");
        output.vector_id = optional_token(component.at("vector_id"), "vector_id");
        output.engine_id = optional_token(component.at("engine_id"), "engine_id");
        output.presence = text_field(component, "presence");
        require_enum(output.presence, {"present", "absent"}, "presence");
        output.required = boolean_field(component, "required");
        const auto scope = text_field(component, "install_scope");
        require_enum(scope, {"prefix", "user", "system", "tops"}, "install_scope");
        if (!component.at("install_root").is_string() ||
            !safe_absolute_path(component.at("install_root").get_ref<const std::string&>())) {
            invalid("lifecycle.invalid_path", "install_root is not a safe absolute path");
        }
        output.activation = text_field(component, "activation");
        require_enum(output.activation, {"inactive", "active", "unmanaged"}, "activation");
        exact_fields(component.at("docking"), {"disposition", "receptor_id"}, "docking");
        output.docking = text_field(component.at("docking"), "disposition");
        require_enum(output.docking, {"undocked", "docked", "unmanaged"}, "docking disposition");
        output.receptor_id = optional_token(component.at("docking").at("receptor_id"), "receptor_id");
        if ((output.docking == "docked") != output.receptor_id.has_value()) {
            invalid("lifecycle.invalid_field", "docking receptor does not match disposition");
        }
        if (component.at("selected_package").is_null()) {
            output.selected_receipt = std::nullopt;
        } else {
            const auto& package = component.at("selected_package");
            exact_fields(package, {"package_id", "version", "receipt_protocol", "receipt_digest"}, "selected package");
            static_cast<void>(text_field(package, "package_id"));
            static_cast<void>(version_field(package, "version"));
            output.selected_receipt_version = receipt_version(text_field(package, "receipt_protocol"));
            parsed.receipt_versions.insert(*output.selected_receipt_version);
            output.selected_receipt = digest_field(package, "receipt_digest");
        }
        if (output.presence == "present" && !output.selected_receipt.has_value()) {
            invalid("lifecycle.invalid_field", "present component requires selected package");
        }
        if (output.presence == "absent" && (output.selected_receipt.has_value() || output.activation == "active" || output.docking == "docked")) {
            invalid("lifecycle.invalid_field", "absent component carries active package state");
        }
        if (!component.at("dependencies").is_array() || component.at("dependencies").size() > max_dependencies) {
            invalid("lifecycle.invalid_dependencies", "dependencies must be a bounded array");
        }
        std::set<std::string> dependency_keys;
        for (const auto& dependency : component.at("dependencies")) {
            exact_fields(dependency, {"target_component_id", "condition", "critical"}, "dependency");
            Dependency parsed_dependency{
                text_field(dependency, "target_component_id"),
                text_field(dependency, "condition"),
                boolean_field(dependency, "critical"),
            };
            require_enum(parsed_dependency.condition, {
                "present", "absent", "installed", "active", "inactive", "docked", "undocked", "compatible"
            }, "dependency condition");
            if (parsed_dependency.target == output.id) {
                invalid("lifecycle.invalid_dependency", "component cannot directly depend on itself");
            }
            if (!dependency_keys.insert(parsed_dependency.target + "|" + parsed_dependency.condition).second) {
                invalid("lifecycle.duplicate_dependency", "dependency is duplicated");
            }
            output.dependencies.push_back(std::move(parsed_dependency));
        }
        exact_fields(component.at("compatibility"), {"required_capabilities", "platform_requirements"}, "component compatibility");
        for (const char* field : {"required_capabilities", "platform_requirements"}) {
            const auto& entries = component.at("compatibility").at(field);
            if (!entries.is_array() || entries.size() > max_capabilities) {
                invalid("lifecycle.invalid_field", std::string(field) + " must be a bounded array");
            }
            std::set<std::string> unique;
            for (const auto& entry : entries) {
                if (!entry.is_string() || !is_token(entry.get_ref<const std::string&>()) ||
                    !unique.insert(entry.get<std::string>()).second) {
                    invalid("lifecycle.invalid_field", std::string(field) + " contains invalid or duplicate entries");
                }
                if (std::string_view(field) == "required_capabilities") {
                    output.required_capabilities.insert(entry.get<std::string>());
                }
            }
        }
        parsed.critical_extension = validate_extensions(component.at("extensions")) || parsed.critical_extension;
        auto normalized_component = component;
        sort_array(normalized_component.at("dependencies"));
        sort_array(normalized_component.at("compatibility").at("required_capabilities"));
        sort_array(normalized_component.at("compatibility").at("platform_requirements"));
        normalize_extensions(normalized_component.at("extensions"));
        output.target_state_digest = engine::tagged_sha256(normalized_component.dump());
        if (!parsed.components.emplace(output.id, std::move(output)).second) {
            invalid("lifecycle.duplicate_component", "desired component identity is duplicated");
        }
    }
    parsed.digest = verify_document_digest(value, "desired_state_digest", normalize_desired);
    return parsed;
}

int receipt_version(std::string_view protocol) {
    if (protocol == "symphony.knowledge.install-receipt.v1") return 1;
    if (protocol == "symphony.knowledge.install-receipt.v2") return 2;
    invalid("lifecycle.unsupported_receipt", "observation contains an unsupported receipt protocol");
}

ParsedObservation parse_observation(const engine::Json& value, std::int64_t deadline_unix_ms) {
    exact_fields(value, {
        "protocol", "format_version", "profile_id", "tops_id", "configured_roots",
        "platform", "binding_registry_digest", "components", "unknown_packages",
        "observed_at", "canonical", "observation_digest"
    }, "observation");
    require_const(value.at("protocol"), "symphony.knowledge.lifecycle-observation.v1", "observation protocol");
    require_const(value.at("format_version"), 1, "observation format version");
    require_const(value.at("canonical"), false, "observation canonical status");
    ParsedObservation parsed{
        text_field(value, "profile_id"),
        text_field(value, "tops_id"),
        digest_field(value, "observation_digest"),
        {},
        optional_digest(value.at("binding_registry_digest"), "binding_registry_digest"),
        {}, {}, {}, false,
    };
    if (!value.at("configured_roots").is_array() || value.at("configured_roots").empty() || value.at("configured_roots").size() > 64U) {
        invalid("lifecycle.invalid_path", "configured_roots must be a non-empty bounded array");
    }
    std::set<std::string> roots;
    for (const auto& root : value.at("configured_roots")) {
        if (!root.is_string() || !safe_absolute_path(root.get_ref<const std::string&>()) ||
            !roots.insert(root.get<std::string>()).second) {
            invalid("lifecycle.invalid_path", "configured root is invalid or duplicated");
        }
    }
    const auto& platform = value.at("platform");
    exact_fields(platform, {
        "os", "kernel_abi", "architecture", "qxctl_identity", "coordinator_identity",
        "provider_availability", "compatibility_digest"
    }, "platform");
    const auto os = text_field(platform, "os");
    require_enum(os, {"linux", "macos"}, "platform os");
    static_cast<void>(text_field(platform, "kernel_abi"));
    static_cast<void>(text_field(platform, "architecture"));
    for (const char* identity_field : {"qxctl_identity", "coordinator_identity"}) {
        const auto& identity = platform.at(identity_field);
        if (identity.is_null() && std::string_view(identity_field) == "coordinator_identity") continue;
        exact_fields(identity, {"component_id", "version", "executable_digest"}, "platform identity");
        static_cast<void>(text_field(identity, "component_id"));
        static_cast<void>(version_field(identity, "version"));
        static_cast<void>(digest_field(identity, "executable_digest"));
    }
    if (!platform.at("provider_availability").is_array() || platform.at("provider_availability").size() > max_capabilities) {
        invalid("lifecycle.invalid_field", "provider availability must be bounded");
    }
    std::set<std::string> providers;
    for (const auto& provider : platform.at("provider_availability")) {
        exact_fields(provider, {"provider_id", "available"}, "provider availability");
        const auto id = text_field(provider, "provider_id");
        static_cast<void>(boolean_field(provider, "available"));
        if (!providers.insert(id).second) invalid("lifecycle.duplicate_provider", "provider is duplicated");
    }
    auto normalized_platform = platform;
    const auto expected_platform = digest_field(normalized_platform, "compatibility_digest");
    normalized_platform.erase("compatibility_digest");
    sort_array(normalized_platform.at("provider_availability"));
    parsed.platform_digest = engine::tagged_sha256(normalized_platform.dump());
    if (parsed.platform_digest != expected_platform) {
        invalid("lifecycle.digest_mismatch", "platform compatibility digest does not match");
    }
    if (!value.at("components").is_array() || value.at("components").size() > max_components) {
        invalid("lifecycle.invalid_components", "observed components must be a bounded array");
    }
    for (const auto& component : value.at("components")) {
        check_deadline(deadline_unix_ms);
        exact_fields(component, {
            "component_id", "component_kind", "module_id", "vector_id", "engine_id",
            "packages", "selected_package_digest", "activation", "docking", "receptor_id",
            "capabilities", "platform_compatibility", "observation_digest"
        }, "observed component");
        ObservedComponent output;
        output.id = text_field(component, "component_id");
        output.kind = text_field(component, "component_kind");
        require_enum(output.kind, {"coordinator", "vector_engine", "module", "adapter", "ui", "service"}, "component_kind");
        output.module_id = text_field(component, "module_id");
        output.vector_id = optional_token(component.at("vector_id"), "vector_id");
        output.engine_id = optional_token(component.at("engine_id"), "engine_id");
        output.selected_receipt = optional_digest(component.at("selected_package_digest"), "selected_package_digest");
        output.activation = text_field(component, "activation");
        require_enum(output.activation, {"inactive", "active", "unknown"}, "activation");
        output.docking = text_field(component, "docking");
        require_enum(output.docking, {"undocked", "docked", "unavailable", "unknown"}, "docking");
        output.receptor_id = optional_token(component.at("receptor_id"), "receptor_id");
        if ((output.docking == "docked") != output.receptor_id.has_value()) {
            invalid("lifecycle.invalid_field", "observed docking receptor does not match disposition");
        }
        output.platform_compatibility = text_field(component, "platform_compatibility");
        require_enum(output.platform_compatibility, {"compatible", "incompatible", "unknown"}, "platform compatibility");
        if (!component.at("packages").is_array() || component.at("packages").empty() || component.at("packages").size() > max_packages) {
            invalid("lifecycle.invalid_packages", "observed packages must be a non-empty bounded array");
        }
        std::set<std::string> package_digests;
        for (const auto& package : component.at("packages")) {
            exact_fields(package, {
                "package_id", "version", "install_root", "receipt_protocol", "receipt_digest",
                "integrity", "entry_points_validated"
            }, "observed package");
            static_cast<void>(text_field(package, "package_id"));
            static_cast<void>(version_field(package, "version"));
            if (!package.at("install_root").is_string() ||
                !safe_absolute_path(package.at("install_root").get_ref<const std::string&>())) {
                invalid("lifecycle.invalid_path", "observed install root is invalid");
            }
            ObservedPackage parsed_package{
                text_field(package, "receipt_protocol"),
                digest_field(package, "receipt_digest"),
                text_field(package, "integrity"),
                boolean_field(package, "entry_points_validated"),
            };
            require_enum(parsed_package.integrity, {"valid", "invalid", "unknown"}, "package integrity");
            parsed.receipt_versions.insert(receipt_version(parsed_package.receipt_protocol));
            if (!package_digests.insert(parsed_package.receipt_digest).second) {
                invalid("lifecycle.duplicate_package", "observed receipt digest is duplicated");
            }
            output.packages.push_back(std::move(parsed_package));
        }
        if (output.selected_receipt.has_value() && !package_digests.contains(*output.selected_receipt)) {
            invalid("lifecycle.invalid_field", "selected package digest is not present in observed packages");
        }
        if ((output.activation == "active" || output.docking == "docked") && !output.selected_receipt.has_value()) {
            invalid("lifecycle.invalid_field", "active or docked component requires a selected package");
        }
        if (!component.at("capabilities").is_array() || component.at("capabilities").size() > max_capabilities) {
            invalid("lifecycle.invalid_field", "observed capabilities must be bounded");
        }
        std::set<std::string> capabilities;
        for (const auto& capability : component.at("capabilities")) {
            if (!capability.is_string() || !is_token(capability.get_ref<const std::string&>()) ||
                !capabilities.insert(capability.get<std::string>()).second) {
                invalid("lifecycle.invalid_field", "observed capability is invalid or duplicated");
            }
        }
        output.capabilities = std::move(capabilities);
        auto normalized_component = component;
        const auto expected_component = digest_field(normalized_component, "observation_digest");
        normalized_component.erase("observation_digest");
        sort_array(normalized_component.at("packages"));
        sort_array(normalized_component.at("capabilities"));
        output.observation_digest = engine::tagged_sha256(normalized_component.dump());
        if (output.observation_digest != expected_component) {
            invalid("lifecycle.digest_mismatch", "component observation digest does not match");
        }
        if (!parsed.components.emplace(output.id, std::move(output)).second) {
            invalid("lifecycle.duplicate_component", "observed component identity is duplicated");
        }
    }
    if (!value.at("unknown_packages").is_array() || value.at("unknown_packages").size() > max_components) {
        invalid("lifecycle.invalid_packages", "unknown packages must be a bounded array");
    }
    for (const auto& package : value.at("unknown_packages")) {
        exact_fields(package, {"install_root", "receipt_path", "reason", "preserved"}, "unknown package");
        if (!package.at("install_root").is_string() || !safe_absolute_path(package.at("install_root").get_ref<const std::string&>())) {
            invalid("lifecycle.invalid_path", "unknown package install root is invalid");
        }
        if (!package.at("receipt_path").is_string() || !engine::is_safe_relative_path(package.at("receipt_path").get_ref<const std::string&>())) {
            invalid("lifecycle.invalid_path", "unknown receipt path is invalid");
        }
        const auto reason = text_field(package, "reason");
        require_enum(reason, {"unsupported_protocol", "invalid_receipt", "ambiguous_identity", "unreadable"}, "unknown package reason");
        require_const(package.at("preserved"), true, "unknown package preservation");
        parsed.critical_unknown = true;
    }
    if (!value.at("observed_at").is_string() || !engine::is_utc_seconds(value.at("observed_at").get_ref<const std::string&>())) {
        invalid("lifecycle.invalid_field", "observed_at must be a normalized UTC timestamp");
    }
    auto stable_inventory = value;
    stable_inventory.erase("observation_digest");
    stable_inventory.erase("observed_at");
    stable_inventory = normalize_observation(std::move(stable_inventory));
    parsed.stable_inventory_digest = engine::tagged_sha256(stable_inventory.dump());
    parsed.digest = verify_document_digest(value, "observation_digest", normalize_observation);
    return parsed;
}

std::set<int> integer_versions(const engine::Json& value, const char* field) {
    if (!value.is_array() || value.empty() || value.size() > 16U) {
        invalid("lifecycle.invalid_client", std::string(field) + " must be a bounded version array");
    }
    std::set<int> output;
    for (const auto& version : value) {
        if (!version.is_number_integer()) invalid("lifecycle.invalid_client", std::string(field) + " contains a non-integer");
        const auto number = version.get<int>();
        if (number < 1 || number > 16 || !output.insert(number).second) {
            invalid("lifecycle.invalid_client", std::string(field) + " contains an invalid or duplicate version");
        }
    }
    return output;
}

engine::Json negotiate(
    const engine::Json& client,
    const std::set<int>& required_receipt_versions) {
    exact_fields(client, {
        "client_id", "client_version", "process_protocols", "desired_state_read_versions",
        "observation_read_versions", "plan_read_versions", "applied_state_read_versions",
        "receipt_read_versions", "capabilities"
    }, "lifecycle client");
    static_cast<void>(text_field(client, "client_id"));
    static_cast<void>(version_field(client, "client_version"));
    std::set<std::string> process_protocols;
    if (!client.at("process_protocols").is_array() || client.at("process_protocols").empty() || client.at("process_protocols").size() > 16U) {
        invalid("lifecycle.invalid_client", "process protocols must be a bounded array");
    }
    for (const auto& protocol : client.at("process_protocols")) {
        if (!protocol.is_string() || !is_token(protocol.get_ref<const std::string&>()) ||
            !process_protocols.insert(protocol.get<std::string>()).second) {
            invalid("lifecycle.invalid_client", "process protocol is invalid or duplicated");
        }
    }
    const auto desired_versions = integer_versions(client.at("desired_state_read_versions"), "desired_state_read_versions");
    const auto observation_versions = integer_versions(client.at("observation_read_versions"), "observation_read_versions");
    const auto plan_versions = integer_versions(client.at("plan_read_versions"), "plan_read_versions");
    const auto applied_versions = integer_versions(client.at("applied_state_read_versions"), "applied_state_read_versions");
    const auto receipt_versions = integer_versions(client.at("receipt_read_versions"), "receipt_read_versions");
    std::set<std::string> client_capabilities;
    if (!client.at("capabilities").is_array() || client.at("capabilities").empty() || client.at("capabilities").size() > max_capabilities) {
        invalid("lifecycle.invalid_client", "client capabilities must be a bounded array");
    }
    for (const auto& capability : client.at("capabilities")) {
        if (!capability.is_string() || !is_token(capability.get_ref<const std::string&>()) ||
            !client_capabilities.insert(capability.get<std::string>()).second) {
            invalid("lifecycle.invalid_client", "client capability is invalid or duplicated");
        }
    }
    std::vector<std::string> missing;
    if (!process_protocols.contains(engine::process_protocol_v1)) missing.push_back("process-v1");
    if (!desired_versions.contains(1)) missing.push_back("desired-state-v1");
    if (!observation_versions.contains(1)) missing.push_back("observation-v1");
    if (!plan_versions.contains(1)) missing.push_back("plan-v1");
    if (!applied_versions.contains(1)) missing.push_back("applied-state-v1");
    auto plan_capabilities = base_required_capabilities;
    for (const auto version : required_receipt_versions) {
        if (!receipt_versions.contains(version)) missing.push_back("receipt-v" + std::to_string(version));
        plan_capabilities.push_back(version == 1 ? "receipt-v1-adapter" : "receipt-v2");
    }
    std::sort(plan_capabilities.begin(), plan_capabilities.end());
    plan_capabilities.erase(std::unique(plan_capabilities.begin(), plan_capabilities.end()), plan_capabilities.end());
    for (const auto& capability : plan_capabilities) {
        if (!client_capabilities.contains(capability)) missing.push_back(capability);
    }
    std::sort(missing.begin(), missing.end());
    missing.erase(std::unique(missing.begin(), missing.end()), missing.end());
    std::vector<int> negotiated_receipts;
    for (const auto version : {1, 2}) {
        if (receipt_versions.contains(version)) negotiated_receipts.push_back(version);
    }
    return engine::Json{
        {"mode", missing.empty() ? "full" : "blocked"},
        {"coordinator_version", engine_version},
        {"desired_state_version", 1},
        {"observation_version", 1},
        {"plan_version", 1},
        {"applied_state_version", 1},
        {"receipt_versions", negotiated_receipts},
        {"required_capabilities", plan_capabilities},
        {"missing_capabilities", missing},
        {"two_way_procedural_compatibility", missing.empty()},
        {"reason", missing.empty() ?
            "all report-only lifecycle protocols, receipt readers, and required capabilities are shared" :
            "one or more report-only lifecycle protocols, receipt readers, or required capabilities are missing"},
    };
}

engine::Json blocker(
    std::string kind,
    std::string component_id,
    std::optional<std::string> action_id,
    bool retryable,
    std::string detail) {
    return engine::Json{
        {"class", std::move(kind)},
        {"component_id", std::move(component_id)},
        {"action_id", action_id.has_value() ? engine::Json(*action_id) : engine::Json(nullptr)},
        {"retryable", retryable},
        {"detail", std::move(detail)},
    };
}

bool identity_matches(const DesiredComponent& desired, const ObservedComponent& observed) {
    return desired.kind == observed.kind && desired.module_id == observed.module_id &&
           desired.vector_id == observed.vector_id && desired.engine_id == observed.engine_id;
}

const ObservedPackage* find_package(const ObservedComponent& component, const std::string& digest) {
    const auto found = std::find_if(component.packages.begin(), component.packages.end(), [&](const ObservedPackage& package) {
        return package.receipt_digest == digest;
    });
    return found == component.packages.end() ? nullptr : &*found;
}

bool condition_satisfied(
    const std::map<std::string, ObservedComponent>& observed,
    const Dependency& dependency) {
    const auto found = observed.find(dependency.target);
    if (dependency.condition == "absent") return found == observed.end();
    if (found == observed.end()) return false;
    const auto& component = found->second;
    if (dependency.condition == "present" || dependency.condition == "installed") return !component.packages.empty();
    if (dependency.condition == "active") return component.activation == "active";
    if (dependency.condition == "inactive") return component.activation == "inactive";
    if (dependency.condition == "docked") return component.docking == "docked";
    if (dependency.condition == "undocked") return component.docking == "undocked";
    if (dependency.condition == "compatible") return component.platform_compatibility == "compatible";
    return false;
}

bool desired_condition_can_satisfy(
    const std::map<std::string, DesiredComponent>& desired,
    const Dependency& dependency) {
    const auto found = desired.find(dependency.target);
    if (dependency.condition == "absent") {
        return found != desired.end() && found->second.presence == "absent";
    }
    if (found == desired.end() || found->second.presence != "present") return false;
    const auto& component = found->second;
    if (dependency.condition == "present" || dependency.condition == "installed") return true;
    if (dependency.condition == "active") return component.activation == "active";
    if (dependency.condition == "inactive") return component.activation == "inactive";
    if (dependency.condition == "docked") return component.docking == "docked";
    if (dependency.condition == "undocked") return component.docking == "undocked";
    if (dependency.condition == "compatible") return true;
    return false;
}

std::string inverse_kind(std::string_view kind) {
    if (kind == "install") return "uninstall";
    if (kind == "uninstall") return "install";
    if (kind == "select") return "deselect";
    if (kind == "deselect") return "select";
    if (kind == "activate") return "deactivate";
    if (kind == "deactivate") return "activate";
    if (kind == "dock") return "undock";
    if (kind == "undock") return "dock";
    return {};
}

std::size_t add_action(
    std::vector<DraftAction>& actions,
    const std::string& component_id,
    std::string kind,
    std::optional<std::string> before,
    const std::string& target_basis,
    std::vector<std::string> artifacts,
    std::vector<std::string> evidence,
    std::optional<std::string> target_receptor = std::nullopt,
    const std::optional<std::size_t>& after = std::nullopt) {
    if (actions.size() >= max_actions) invalid("lifecycle.action_limit", "lifecycle action limit exceeded");
    std::sort(artifacts.begin(), artifacts.end());
    std::sort(evidence.begin(), evidence.end());
    DraftAction action{
        component_id,
        std::move(kind),
        {},
        "forward",
        std::move(before),
        {},
        std::move(target_receptor),
        std::move(artifacts),
        std::move(evidence),
        {}, {}, {}, {}, std::nullopt,
    };
    action.inverse_kind = inverse_kind(action.kind);
    if (action.inverse_kind.empty()) action.direction = "neutral";
    if (after.has_value()) action.prerequisites.push_back(*after);
    action.target_state_digest = engine::tagged_sha256(
        target_basis + "\nkind=" + action.kind + "\nreceptor=" + action.target_receptor_id.value_or("absent"));
    action.semantic_key = component_id + "\n" + action.kind + "\n" + action.direction + "\n" +
        action.expected_before.value_or("absent") + "\ntarget=" + action.target_state_digest;
    for (const auto& digest : action.artifact_digests) action.semantic_key += "\nartifact=" + digest;
    for (const auto& item : action.expected_evidence) action.semantic_key += "\nevidence=" + item;
    actions.push_back(std::move(action));
    return actions.size() - 1U;
}

void add_prerequisite(DraftAction& action, std::size_t prerequisite) {
    if (std::find(action.prerequisites.begin(), action.prerequisites.end(), prerequisite) == action.prerequisites.end()) {
        action.prerequisites.push_back(prerequisite);
    }
}

void assign_action_ids(std::vector<DraftAction>& actions) {
    for (auto& action : actions) {
        std::vector<std::string> prerequisite_semantics;
        for (const auto index : action.prerequisites) {
            if (index >= actions.size()) invalid("lifecycle.internal", "action prerequisite is out of range");
            prerequisite_semantics.push_back(actions[index].semantic_key);
        }
        std::sort(prerequisite_semantics.begin(), prerequisite_semantics.end());
        auto basis = action.semantic_key;
        for (const auto& prerequisite : prerequisite_semantics) basis += "\nprerequisite=" + prerequisite;
        action.action_id = "lifecycle-action:" + engine::sha256_hex(basis);
        if (!action.inverse_kind.empty()) {
            action.inverse_action_id = "lifecycle-action:" + engine::sha256_hex(
                "inverse\n" + action.semantic_key + "\nkind=" + action.inverse_kind);
        }
    }
    std::set<std::string> identities;
    for (const auto& action : actions) {
        if (!identities.insert(action.action_id).second) {
            invalid("lifecycle.duplicate_action", "planner produced duplicate action identity");
        }
    }
}

std::set<std::size_t> cyclic_actions(const std::vector<DraftAction>& actions) {
    std::vector<int> index(actions.size(), -1);
    std::vector<int> low(actions.size(), -1);
    std::vector<bool> on_stack(actions.size(), false);
    std::vector<std::size_t> stack;
    std::set<std::size_t> cyclic;
    int next = 0;
    std::function<void(std::size_t)> visit = [&](const std::size_t node) {
        index[node] = next;
        low[node] = next;
        ++next;
        stack.push_back(node);
        on_stack[node] = true;
        for (const auto prerequisite : actions[node].prerequisites) {
            if (index[prerequisite] == -1) {
                visit(prerequisite);
                low[node] = std::min(low[node], low[prerequisite]);
            } else if (on_stack[prerequisite]) {
                low[node] = std::min(low[node], index[prerequisite]);
            }
        }
        if (low[node] != index[node]) return;
        std::vector<std::size_t> component;
        while (true) {
            const auto current = stack.back();
            stack.pop_back();
            on_stack[current] = false;
            component.push_back(current);
            if (current == node) break;
        }
        const bool self_loop = component.size() == 1U &&
            std::find(actions[node].prerequisites.begin(), actions[node].prerequisites.end(), node) != actions[node].prerequisites.end();
        if (component.size() > 1U || self_loop) cyclic.insert(component.begin(), component.end());
    };
    for (std::size_t node = 0; node < actions.size(); ++node) {
        if (index[node] == -1) visit(node);
    }
    return cyclic;
}

engine::Json scheduler_truth() {
    return engine::Json{
        {"algorithm", "dependency_ready_set_v1"},
        {"dynamic_replanning", true},
        {"directionality", "forward_and_inverse"},
        {"tie_break", "lexicographic_action_id"},
        {"safety_phase_order", engine::Json::array({
            "lock", "observe", "authorize", "compare_and_swap", "act", "verify", "audit"
        })},
        {"cycle_policy", "block_cyclic_component_continue_unrelated"},
        {"max_actions", 4096},
        {"max_replans_per_transaction", 256},
        {"max_attempts_per_action", 8},
    };
}

engine::Json build_plan(
    const ParsedDesired& desired,
    const ParsedObservation& observation,
    const std::optional<std::string>& prior_applied,
    engine::Json compatibility,
    std::int64_t deadline_unix_ms) {
    std::vector<engine::Json> fatal_blockers;
    if (desired.profile_id != observation.profile_id || desired.tops_id != observation.tops_id) {
        fatal_blockers.push_back(blocker(
            "critical_state_unknown", "lifecycle", std::nullopt, false,
            "desired and observed profile or TOPS identity does not match"));
    }
    if (desired.critical_extension) {
        fatal_blockers.push_back(blocker(
            "critical_state_unknown", "lifecycle", std::nullopt, false,
            "desired state contains an unknown critical extension"));
    }
    if (observation.critical_unknown) {
        fatal_blockers.push_back(blocker(
            "critical_state_unknown", "lifecycle", std::nullopt, false,
            "observation contains preserved unknown or unreadable package evidence"));
    }
    if (compatibility.at("mode") != "full") {
        fatal_blockers.push_back(blocker(
            "compatibility_blocked", "lifecycle", std::nullopt, true,
            "client and coordinator do not share every required report-only lifecycle capability"));
    }
    if (fatal_blockers.empty()) {
        for (const auto& [component_id, desired_component] : desired.components) {
            const auto found = observation.components.find(component_id);
            if (found == observation.components.end()) continue;
            const auto& observed_component = found->second;
            if (!identity_matches(desired_component, observed_component)) {
                fatal_blockers.push_back(blocker(
                    "critical_state_unknown", component_id, std::nullopt, false,
                    "desired and observed component identity fields disagree"));
                continue;
            }
            if (observed_component.activation == "unknown" || observed_component.docking == "unknown" ||
                observed_component.platform_compatibility == "unknown") {
                fatal_blockers.push_back(blocker(
                    "critical_state_unknown", component_id, std::nullopt, false,
                    "selected component state or platform compatibility is unknown"));
                continue;
            }
            if (observed_component.selected_receipt.has_value()) {
                const auto* selected = find_package(observed_component, *observed_component.selected_receipt);
                if (selected == nullptr || selected->integrity != "valid" || !selected->entry_points_validated) {
                    fatal_blockers.push_back(blocker(
                        "integrity_fatal", component_id, std::nullopt, false,
                        "the currently selected package lacks valid receipt or entry-point evidence"));
                    continue;
                }
            }
            if (desired_component.presence == "present" && desired_component.selected_receipt.has_value()) {
                const auto* selected = find_package(observed_component, *desired_component.selected_receipt);
                if (selected != nullptr && (selected->integrity != "valid" || !selected->entry_points_validated)) {
                    fatal_blockers.push_back(blocker(
                        "integrity_fatal", component_id, std::nullopt, false,
                        "the exact desired package lacks valid receipt or entry-point evidence"));
                } else if (selected != nullptr && desired_component.selected_receipt_version.has_value() &&
                           receipt_version(selected->receipt_protocol) != *desired_component.selected_receipt_version) {
                    fatal_blockers.push_back(blocker(
                        "integrity_fatal", component_id, std::nullopt, false,
                        "the exact desired package digest is associated with a different receipt protocol version"));
                }
            }
        }
    }

    const std::string binding = observation.binding_registry_digest.value_or("not_applicable");
    std::string capability_basis;
    for (const auto& capability : supported_capabilities) capability_basis += capability + "\n";
    const auto capability_digest = engine::tagged_sha256(capability_basis);
    const auto observation_key = engine::tagged_sha256(
        desired.digest + "\n" + observation.stable_inventory_digest + "\n" + binding + "\n" +
        desired.profile_id + "\n" + desired.tops_id + "\n" + observation.platform_digest +
        "\n" + capability_digest);
    const auto transaction_id = "lifecycle-transaction:" + engine::sha256_hex(
        observation_key + "\n" + prior_applied.value_or("absent"));

    std::vector<DraftAction> actions;
    engine::Json advisories = engine::Json::array();
    std::map<std::string, std::pair<std::size_t, std::size_t>> component_action_range;
    if (fatal_blockers.empty()) {
        for (const auto& [component_id, component] : desired.components) {
            check_deadline(deadline_unix_ms);
            const auto observed_found = observation.components.find(component_id);
            const ObservedComponent* observed = observed_found == observation.components.end() ? nullptr : &observed_found->second;
            const auto start = actions.size();
            std::optional<std::size_t> previous;
            if (component.presence == "present") {
                const ObservedPackage* selected_package = observed == nullptr ? nullptr : find_package(*observed, *component.selected_receipt);
                if (selected_package == nullptr) {
                    const auto index = add_action(actions, component_id, "install", std::nullopt,
                        component.target_state_digest, {*component.selected_receipt},
                        {"receipt_integrity", "platform_compatibility"});
                    actions[index].blockers.push_back(blocker(
                        "dependency_wait", component_id, std::nullopt, true,
                        "the exact desired package is not present in the observation"));
                    previous = index;
                } else {
                    const bool selection_change = observed->selected_receipt != component.selected_receipt;
                    const bool receptor_change = component.docking == "docked" &&
                        (observed->docking != "docked" || observed->receptor_id != component.receptor_id);
                    const bool must_undock = observed->docking == "docked" &&
                        (selection_change || component.docking == "undocked" || receptor_change);
                    auto effective_activation = observed->activation;
                    if (must_undock) {
                        previous = add_action(actions, component_id, "undock", observed->observation_digest,
                            component.target_state_digest, {*component.selected_receipt},
                            {"authorization_required_at_apply", "maestro_receptor_evidence"},
                            std::nullopt, previous);
                    }
                    if (selection_change && effective_activation == "active") {
                        previous = add_action(actions, component_id,
                            "deactivate", observed->observation_digest, component.target_state_digest,
                            {*component.selected_receipt},
                            {"authorization_required_at_apply", "component_state_digest"},
                            std::nullopt, previous);
                        effective_activation = "inactive";
                    }
                    if (selection_change) {
                        previous = add_action(actions, component_id, "select", observed->observation_digest,
                            component.target_state_digest, {*component.selected_receipt},
                            {"component_state_digest", "platform_compatibility", "receipt_integrity", "required_capabilities"},
                            std::nullopt, previous);
                    }
                    const auto target_activation = component.activation == "unmanaged" ? observed->activation : component.activation;
                    if (effective_activation != target_activation) {
                        previous = add_action(actions, component_id,
                            target_activation == "active" ? "activate" : "deactivate",
                            observed->observation_digest, component.target_state_digest,
                            {*component.selected_receipt},
                            {"authorization_required_at_apply", "component_state_digest"},
                            std::nullopt, previous);
                    }
                    if (component.docking == "docked" && (receptor_change || must_undock)) {
                        if (observed->docking == "unavailable" && component.docking == "docked") {
                            const auto index = add_action(actions, component_id, "dock", observed->observation_digest,
                                component.target_state_digest, {*component.selected_receipt},
                                {"maestro_receptor_evidence"}, component.receptor_id, previous);
                            actions[index].blockers.push_back(blocker(
                                "compatibility_blocked", component_id, std::nullopt, true,
                                "the selected Maestro receptor is not available"));
                            previous = index;
                        } else {
                            previous = add_action(actions, component_id, "dock",
                                observed->observation_digest, component.target_state_digest,
                                {*component.selected_receipt},
                                {"authorization_required_at_apply", "maestro_receptor_evidence"},
                                component.receptor_id, previous);
                        }
                    } else if (component.docking == "unmanaged" && must_undock && observed->receptor_id.has_value()) {
                        previous = add_action(actions, component_id, "dock",
                            observed->observation_digest, component.target_state_digest,
                            {*component.selected_receipt},
                            {"authorization_required_at_apply", "maestro_receptor_evidence"},
                            observed->receptor_id, previous);
                    }
                }
                const bool exact_package_selected = observed != nullptr &&
                    observed->selected_receipt == component.selected_receipt;
                if (exact_package_selected && observed->platform_compatibility == "incompatible") {
                    if (actions.size() == start) {
                        previous = add_action(actions, component_id, "verify", observed->observation_digest,
                            component.target_state_digest, {*component.selected_receipt},
                            {"platform_compatibility"});
                    }
                    actions[start].blockers.push_back(blocker(
                        "compatibility_blocked", component_id, std::nullopt, true,
                        "the selected component is incompatible with the observed platform"));
                }
                if (exact_package_selected) {
                    std::vector<std::string> missing_capabilities;
                    std::set_difference(
                        component.required_capabilities.begin(), component.required_capabilities.end(),
                        observed->capabilities.begin(), observed->capabilities.end(),
                        std::back_inserter(missing_capabilities));
                    if (!missing_capabilities.empty()) {
                        if (actions.size() == start) {
                            previous = add_action(actions, component_id, "verify", observed->observation_digest,
                                component.target_state_digest, {*component.selected_receipt},
                                {"required_capabilities"});
                        }
                        std::string detail = "observed component is missing required capabilities:";
                        for (const auto& capability : missing_capabilities) detail += " " + capability;
                        actions[start].blockers.push_back(blocker(
                            "compatibility_blocked", component_id, std::nullopt, true, std::move(detail)));
                    }
                }
            } else if (observed != nullptr) {
                if (observed->docking == "docked") {
                    previous = add_action(actions, component_id, "undock", observed->observation_digest,
                        component.target_state_digest, {},
                        {"authorization_required_at_apply", "maestro_receptor_evidence"},
                        std::nullopt, previous);
                }
                if (observed->activation == "active") {
                    previous = add_action(actions, component_id, "deactivate", observed->observation_digest,
                        component.target_state_digest, {},
                        {"authorization_required_at_apply", "component_state_digest"},
                        std::nullopt, previous);
                }
                if (observed->selected_receipt.has_value()) {
                    previous = add_action(actions, component_id, "deselect", observed->observation_digest,
                        component.target_state_digest, {*observed->selected_receipt},
                        {"authorization_required_at_apply", "receipt_integrity"},
                        std::nullopt, previous);
                }
                std::vector<std::string> receipts;
                for (const auto& package : observed->packages) receipts.push_back(package.receipt_digest);
                previous = add_action(actions, component_id, "uninstall", observed->observation_digest,
                    component.target_state_digest, std::move(receipts),
                    {"authorization_required_at_apply", "receipt_owned_files"},
                    std::nullopt, previous);
            }
            if (actions.size() > start) component_action_range[component_id] = {start, actions.size() - 1U};
        }
        for (const auto& [component_id, observed] : observation.components) {
            if (desired.components.contains(component_id)) continue;
            const auto index = add_action(actions, component_id, "preserve", observed.observation_digest,
                observed.observation_digest, {}, {"unmanaged_component_preservation"});
            component_action_range[component_id] = {index, index};
        }
        for (const auto& [component_id, component] : desired.components) {
            for (const auto& dependency : component.dependencies) {
                if (condition_satisfied(observation.components, dependency)) continue;
                if (!dependency.critical) {
                    advisories.push_back(engine::Json{
                        {"class", "noncritical_dependency_unsatisfied"},
                        {"component_id", component_id},
                        {"target_component_id", dependency.target},
                        {"condition", dependency.condition},
                        {"detail", "noncritical dependency is not satisfied by the current observation"},
                    });
                    continue;
                }
                auto range = component_action_range.find(component_id);
                if (range == component_action_range.end()) {
                    const auto observed = observation.components.find(component_id);
                    const auto index = add_action(actions, component_id, "verify",
                        observed == observation.components.end() ? std::nullopt : std::optional<std::string>(observed->second.observation_digest),
                        component.target_state_digest, {}, {"dependency_state"});
                    range = component_action_range.emplace(component_id, std::make_pair(index, index)).first;
                }
                const auto target = component_action_range.find(dependency.target);
                if (target == component_action_range.end() ||
                    !desired_condition_can_satisfy(desired.components, dependency)) {
                    actions[range->second.first].blockers.push_back(blocker(
                        "dependency_wait", component_id, std::nullopt, true,
                        "dependency " + dependency.target + " does not satisfy " + dependency.condition));
                } else {
                    add_prerequisite(actions[range->second.first], target->second.second);
                }
            }
        }
    }

    assign_action_ids(actions);
    for (auto& action : actions) {
        for (auto& item : action.blockers) item["action_id"] = action.action_id;
    }
    const auto cycles = cyclic_actions(actions);
    for (const auto index : cycles) {
        actions[index].blockers.push_back(blocker(
            "cycle_detected", actions[index].component_id, actions[index].action_id, false,
            "the unresolved action dependency graph contains a cycle"));
    }

    engine::Json action_values = engine::Json::array();
    engine::Json ready = engine::Json::array();
    engine::Json deferred = engine::Json::array();
    engine::Json blocked = engine::Json::array();
    std::vector<std::size_t> order(actions.size());
    for (std::size_t index = 0; index < actions.size(); ++index) order[index] = index;
    std::sort(order.begin(), order.end(), [&](const auto left, const auto right) {
        return actions[left].action_id < actions[right].action_id;
    });
    for (const auto index : order) {
        auto& action = actions[index];
        std::vector<std::string> prerequisite_ids;
        for (const auto prerequisite : action.prerequisites) prerequisite_ids.push_back(actions[prerequisite].action_id);
        std::sort(prerequisite_ids.begin(), prerequisite_ids.end());
        std::sort(action.blockers.begin(), action.blockers.end(), [](const engine::Json& left, const engine::Json& right) {
            return left.dump() < right.dump();
        });
        const std::string disposition = !action.blockers.empty() ? "blocked" :
            prerequisite_ids.empty() ? "ready" : "waiting";
        if (disposition == "ready") ready.push_back(action.action_id);
        else if (disposition == "waiting") deferred.push_back(action.action_id);
        else blocked.push_back(action.action_id);
        action_values.push_back(engine::Json{
            {"action_id", action.action_id},
            {"component_id", action.component_id},
            {"kind", action.kind},
            {"direction", action.direction},
            {"prerequisite_action_ids", prerequisite_ids},
            {"inverse_action_id", action.inverse_action_id.has_value() ? engine::Json(*action.inverse_action_id) : engine::Json(nullptr)},
            {"expected_before_digest", action.expected_before.has_value() ? engine::Json(*action.expected_before) : engine::Json(nullptr)},
            {"target_state_digest", action.target_state_digest},
            {"target_receptor_id", action.target_receptor_id.has_value() ? engine::Json(*action.target_receptor_id) : engine::Json(nullptr)},
            {"expected_artifact_digests", action.artifact_digests},
            {"expected_evidence", action.expected_evidence},
            {"disposition", disposition},
            {"blockers", action.blockers},
        });
    }
    sort_array(ready);
    sort_array(deferred);
    sort_array(blocked);
    sort_array(advisories);
    std::sort(fatal_blockers.begin(), fatal_blockers.end(), [](const engine::Json& left, const engine::Json& right) {
        return left.dump() < right.dump();
    });

    engine::Json plan{
        {"protocol", "symphony.knowledge.lifecycle-plan.v1"},
        {"format_version", 1},
        {"transaction_id", transaction_id},
        {"revision", 1},
        {"previous_plan_digest", nullptr},
        {"desired_state_digest", desired.digest},
        {"observation_digest", observation.digest},
        {"observation_key", observation_key},
        {"prior_applied_state_digest", prior_applied.has_value() ? engine::Json(*prior_applied) : engine::Json(nullptr)},
        {"compatibility", std::move(compatibility)},
        {"scheduler", scheduler_truth()},
        {"actions", std::move(action_values)},
        {"ready_action_ids", std::move(ready)},
        {"deferred_action_ids", std::move(deferred)},
        {"blocked_action_ids", std::move(blocked)},
        {"advisories", std::move(advisories)},
        {"fatal_blockers", std::move(fatal_blockers)},
        {"apply_authorized", false},
        {"canonical", false},
        {"plan_digest", nullptr},
    };
    plan["plan_digest"] = engine::tagged_sha256([&] {
        auto value = plan;
        value.erase("plan_digest");
        return value.dump();
    }());
    return plan;
}

}

engine::Json lifecycle_capabilities() {
    return engine::Json{
        {"operation", "lifecycle_plan"},
        {"state", "implemented_report_only"},
        {"command_protocol", "symphony.knowledge.lifecycle-plan-command.v1"},
        {"desired_state_versions", engine::Json::array({1})},
        {"observation_versions", engine::Json::array({1})},
        {"plan_versions", engine::Json::array({1})},
        {"applied_state_versions", engine::Json::array({1})},
        {"receipt_read_versions", engine::Json::array({1, 2})},
        {"required_capabilities", base_required_capabilities},
        {"supported_capabilities", supported_capabilities},
        {"two_way_procedural_compatibility", true},
        {"dynamic_replanning", true},
        {"configured_root_discovery", false},
        {"persistence_enabled", false},
        {"action_execution_enabled", false},
        {"apply_authorized", false},
    };
}

engine::Json handle_lifecycle_plan(const engine::Request& request) {
    return build_lifecycle_plan(request.payload, request.deadline_unix_ms);
}

engine::Json build_lifecycle_plan(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    check_deadline(deadline_unix_ms);
    exact_fields(payload, {
        "protocol", "operation", "desired_state", "observation",
        "prior_applied_state_digest", "client"
    }, "lifecycle plan command");
    require_const(payload.at("protocol"),
        "symphony.knowledge.lifecycle-plan-command.v1", "lifecycle command protocol");
    require_const(payload.at("operation"), "lifecycle_plan", "lifecycle operation");
    const auto desired = parse_desired(payload.at("desired_state"), deadline_unix_ms);
    check_deadline(deadline_unix_ms);
    const auto observation = parse_observation(payload.at("observation"), deadline_unix_ms);
    const auto prior = optional_digest(payload.at("prior_applied_state_digest"), "prior_applied_state_digest");
    auto receipt_versions = desired.receipt_versions;
    receipt_versions.insert(observation.receipt_versions.begin(), observation.receipt_versions.end());
    auto compatibility = negotiate(payload.at("client"), receipt_versions);
    check_deadline(deadline_unix_ms);
    return build_plan(desired, observation, prior, std::move(compatibility), deadline_unix_ms);
}

std::string lifecycle_stable_inventory_digest(
    const engine::Json& observation,
    std::int64_t deadline_unix_ms) {
    return parse_observation(observation, deadline_unix_ms).stable_inventory_digest;
}

}
