#include "volume_policy.hpp"
#include <iostream>

namespace {
const std::string token = R"({"type":"systemd-fido2","keyslots":["1"],"fido2-rp":"io.systemd.cryptsetup","fido2-clientPin-required":true,"fido2-up-required":true,"fido2-salt":"salt","fido2-credential":"id"})";
const std::string mount = "13 10 253:8 / /var/lib/symphony/nodevault rw - ext4 /dev/mapper/symphonynodevault rw";
std::string replace(std::string value, std::string_view from, std::string_view to) {
    value.replace(value.find(from), from.size(), to);
    return value;
}
std::string header(const std::string& slots, const std::string& tokens) {
    return "{\"keyslots\":{" + slots + "},\"tokens\":{" + tokens + "}}\n";
}
template<class Test> void refused(Test run) {
    try { run(); } catch (const std::runtime_error&) { return; }
    throw std::logic_error("unexpected acceptance");
}
}
int main() {
    try {
        const std::string first = "\"1\":{}";
        const auto check = [&](const std::string& t) { volume::check_slots(header(first, "\"0\":" + t)); };
        check(token);
        const auto two = first + ",\"2\":{}";
        volume::check_slots(header(two, "\"0\":" + token + ",\"1\":" + replace(token, "[\"1\"]", "[\"2\"]")));
        refused([&] { volume::check_slots(header(two, "\"0\":" + token)); });
        for (const auto& invalid : {
                 replace(token, "systemd-fido2", "systemd-tpm2"), replace(token, "required\":true", "required\":\"true\""),
                 replace(token, "up-required\":true", "up-required\":false"), replace(token, "io.systemd.cryptsetup", "other"),
                 replace(token, "[\"1\"]", "[]"), replace(token, "[\"1\"]", "[\"2\"]"), replace(token, "\"id\"", "\"\"")}) {
            refused([&] { check(invalid); });
        }
        refused([&] { volume::check_slots(header(first, "\"0\":" + token + ",\"1\":" + token)); });
        refused([&] { volume::check_slots(header(first, "\"0\":" + token) + "bad"); });
        std::istringstream good(mount);
        if (volume::mounted_device(good) != "253:8") return 1;
        for (const auto& text : {std::string(), replace(mount, "nodevault rw", "elsewhere rw"),
                 replace(mount, " / /var", " /subdir /var"), replace(mount, "ext4", "tmpfs"),
                 mount + "\n" + mount, mount + "\n" + replace(mount, "nodevault rw", "nodevault/.local rw")}) {
            refused([&] { std::istringstream invalid(text); volume::mounted_device(invalid); });
        }
        std::cout << "PASS hardware mount and enrollment policy\n";
    } catch (const std::exception& err) {
        std::cerr << err.what() << '\n';
        return 1;
    }
}
