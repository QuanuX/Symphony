#pragma once

#include <json-c/json.h>
#include <charconv>
#include <limits>
#include <memory>
#include <set>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>

namespace volume {
inline constexpr std::string_view home = "/var/lib/symphony/nodevault";
inline void require(bool ok, const char* reason) {
    if (!ok) throw std::runtime_error(reason);
}
inline unsigned decimal(std::string_view value) {
    unsigned parsed = 0;
    const auto [end, err] = std::from_chars(value.data(), value.data() + value.size(), parsed);
    require(!value.empty() && err == std::errc{} && end == value.data() + value.size(), "invalid integer");
    return parsed;
}
inline void check_device(std::string_view s) {
    const auto split = s.find(':');
    require(split != s.npos, "invalid device");
    decimal(s.substr(0, split));
    decimal(s.substr(split + 1));
}
inline std::string mounted_device(std::istream& info) {
    std::string line, selected;
    while (std::getline(info, line)) {
        const auto separator = line.find(" - ");
        require(separator != line.npos, "invalid mount information");
        std::istringstream left(line.substr(0, separator)), right(line.substr(separator + 3));
        std::string id, parent, device, root, path, filesystem;
        require(bool(left >> id >> parent >> device >> root >> path) && bool(right >> filesystem), "invalid mount information");
        require(!path.starts_with(std::string(home) + "/"), "unexpected nested mount");
        if (path != home) continue;
        require(selected.empty() && root == "/" && filesystem == "ext4", "unexpected filesystem");
        check_device(device);
        selected = device;
    }
    require(info.eof() && !selected.empty(), "dedicated volume is not mounted");
    return selected;
}

inline json_object* field(json_object* object, const char* key, json_type expected) {
    json_object* value = nullptr;
    require(json_object_object_get_ex(object, key, &value) && json_object_is_type(value, expected), "invalid enrollment");
    return value;
}
inline std::string_view string_value(json_object* value) {
    return {json_object_get_string(value), static_cast<std::size_t>(json_object_get_string_len(value))};
}
inline void check_slots(const std::string& metadata) {
    require(!metadata.empty() && metadata.size() <= (4U << 20), "invalid metadata");
    using Parser = std::unique_ptr<json_tokener, decltype(&json_tokener_free)>;
    Parser parser(json_tokener_new(), json_tokener_free);
    require(bool(parser), "parser unavailable");
    json_tokener_set_flags(parser.get(), JSON_TOKENER_STRICT | JSON_TOKENER_VALIDATE_UTF8);
    using Json = std::unique_ptr<json_object, decltype(&json_object_put)>;
    Json header(json_tokener_parse_ex(parser.get(), metadata.data(), static_cast<int>(metadata.size())), json_object_put);
    require(json_tokener_get_error(parser.get()) == json_tokener_success &&
            json_tokener_get_parse_end(parser.get()) == metadata.size() &&
            json_object_is_type(header.get(), json_type_object), "invalid metadata");
    auto* slots = field(header.get(), "keyslots", json_type_object);
    auto* tokens = field(header.get(), "tokens", json_type_object);
    require(json_object_object_length(slots) > 0, "missing enrollment");
    std::set<std::string> covered;
    json_object_object_foreach(tokens, name, token) {
        (void)name;
        require(json_object_is_type(token, json_type_object), "invalid enrollment");
        require(string_value(field(token, "type", json_type_string)) == "systemd-fido2", "unexpected unlock method");
        require(json_object_get_boolean(field(token, "fido2-clientPin-required", json_type_boolean)) &&
                json_object_get_boolean(field(token, "fido2-up-required", json_type_boolean)) &&
                string_value(field(token, "fido2-rp", json_type_string)) == "io.systemd.cryptsetup" &&
                !string_value(field(token, "fido2-salt", json_type_string)).empty() &&
                !string_value(field(token, "fido2-credential", json_type_string)).empty(), "FIDO2 enrollment requires PIN and user presence");
        auto* bindings = field(token, "keyslots", json_type_array);
        require(json_object_array_length(bindings) == 1, "unexpected slot binding");
        auto* slot = json_object_array_get_idx(bindings, 0);
        require(json_object_is_type(slot, json_type_string), "unexpected slot binding");
        const auto id = string_value(slot);
        decimal(id); // LUKS slot indices are decimal. Also excludes embedded NULs.
        json_object* entry = nullptr;
        require(json_object_object_get_ex(slots, std::string(id).c_str(), &entry) && covered.emplace(id).second,
                "unexpected slot binding");
    }
    require(covered.size() == static_cast<std::size_t>(json_object_object_length(slots)), "extra unlock slot");
}
} // namespace volume
