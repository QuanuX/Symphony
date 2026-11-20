#include <symphony/ssiag/linux_secret_service.hpp>

#include <sys/random.h>
#include <libsecret/secret.h>
#include <algorithm>
#include <array>
#include <iostream>
#include <memory>
#include <stdexcept>
#include <string_view>

namespace provider = symphony::ssiag::linux_secret_service;
using provider::Status;

void expect(bool condition, const char* check) {
    if (!condition) throw std::runtime_error(check);
}
std::string uuid() {
    std::unique_ptr<gchar, decltype(&g_free)> id(g_uuid_string_random(), g_free);
    return id.get();
}
// Manufacture an ambiguous match only in this disposable, private session.
// External desktop clients can create identical lookup attributes.
void add_duplicate(const provider::Item& item, provider::Bytes bytes, GCancellable* cancel) {
    using Obj = std::unique_ptr<GObject, decltype(&g_object_unref)>;
    Obj service(G_OBJECT(secret_service_get_sync(SECRET_SERVICE_OPEN_SESSION, cancel, nullptr)), g_object_unref);
    expect(service != nullptr, "fixture service");
    Obj collection(G_OBJECT(secret_collection_for_alias_sync(SECRET_SERVICE(service.get()), "default",
                            SECRET_COLLECTION_NONE, cancel, nullptr)), g_object_unref);
    expect(collection && item.scope.collection_path == g_dbus_proxy_get_object_path(G_DBUS_PROXY(collection.get())), "fixture collection");
    using Attrs = std::unique_ptr<GHashTable, decltype(&g_hash_table_unref)>;
    Attrs attrs(g_hash_table_new(g_str_hash, g_str_equal), g_hash_table_unref);
    auto attr = [&](const char* name, const char* value) {
        g_hash_table_insert(attrs.get(), const_cast<char*>(name), const_cast<char*>(value));
    };
    attr("xdg:schema", "org.quanux.Symphony.SSIAG.GenericCredential.v1");
    attr("tops", item.scope.tops_id.c_str()); attr("reference", item.scope.reference.c_str()); attr("generation", item.generation.c_str());
    std::unique_ptr<SecretValue, decltype(&secret_value_unref)> value(secret_value_new(
        reinterpret_cast<const gchar*>(bytes.data()), bytes.size(), "application/octet-stream"), secret_value_unref);
    Obj twin(G_OBJECT(secret_item_create_sync(SECRET_COLLECTION(collection.get()), nullptr, attrs.get(), "Fixture duplicate",
                      value.get(), SECRET_ITEM_CREATE_NONE, cancel, nullptr)), g_object_unref);
    expect(twin != nullptr, "duplicate fixture");
}
int main(int argc, char** argv) {
    // This is a fixture executable only. Restrict it to the bus path provided
    // by private-session.sh before opening a collection, even on manual runs.
    constexpr std::string_view prefix = "unix:path=/tmp/symphony-secret-service.";
    const char* address = g_getenv("DBUS_SESSION_BUS_ADDRESS");
    if (argc != 2 || !address || !std::string_view(address).starts_with(prefix)) return 2;
    const std::string_view mode(argv[1]);
    try {
        provider::Scope scope{"/org/freedesktop/secrets/collection/login", uuid(), uuid()};
        std::unique_ptr<GCancellable, decltype(&g_object_unref)> cancel(g_cancellable_new(), g_object_unref);
        std::array<std::byte, 32> data{};
        expect(getrandom(data.data(), data.size(), 0) == static_cast<ssize_t>(data.size()), "fixture entropy");
        data[1] = std::byte{0}; // Verify embedded NUL survives the Secret Service transport.
        if (mode == "cached-bus") {
            // Prime GLib's shared bus against the healthy fixture, then select
            // an absent socket. Reusing the cached connection would succeed.
            std::unique_ptr<GDBusConnection, decltype(&g_object_unref)> cached(
                g_bus_get_sync(G_BUS_TYPE_SESSION, cancel.get(), nullptr), g_object_unref);
            expect(cached != nullptr, "prime shared fixture bus");
            const std::string missing_socket = std::string(address) + "-missing";
            expect(g_setenv("DBUS_SESSION_BUS_ADDRESS", missing_socket.c_str(), TRUE), "switch fixture endpoint");
            expect(provider::provision(scope, data, cancel.get()).status == Status::unavailable,
                   "selected endpoint unavailable despite cached live service");
            return 0;
        }
        const auto result = provider::provision(scope, data, cancel.get());
        if (mode == "locked") { expect(result.status == Status::locked, "locked collection refusal"); return 0; }
        if (mode == "absent") { expect(result.status == Status::unavailable, "missing service refusal"); return 0; }
        if (mode == "ambiguous") {
            expect(result.status == Status::ok, "prepare ambiguous fixture");
            add_duplicate(result.item, data, cancel.get());
            unsigned calls = 0;
            expect(provider::use(result.item, 32, cancel.get(), [&](provider::Bytes) { ++calls; }) == Status::ambiguous && calls == 0, "duplicate lookup refusal");
            expect(provider::retire(result.item, cancel.get()) == Status::ambiguous, "duplicate retire refusal");
            return 0;
        }
        expect(mode == "lifecycle" && result.status == Status::ok, "create fixture credential");
        const auto item = result.item;
        unsigned calls = 0;
        auto use = [&](const provider::Item& selected, std::size_t limit) {
            return provider::use(selected, limit, cancel.get(), [&](provider::Bytes value) {
                expect(std::ranges::equal(value, data), "fixture credential match");
                ++calls;
            });
        };
        expect(use(item, data.size()) == Status::ok && calls == 1, "use exact generation");
        expect(use(item, 3) == Status::invalid && calls == 1, "byte ceiling refusal");
        auto other = item;
        other.scope.tops_id = uuid();
        expect(use(other, 32) == Status::not_found && calls == 1, "TOPS isolation");
        other = item;
        other.scope.collection_path = "/org/freedesktop/secrets/collection/session";
        expect(use(other, 32) == Status::not_found && calls == 1, "collection isolation");
        other = item;
        other.generation = uuid();
        expect(provider::retire(other, cancel.get()) == Status::not_found, "wrong generation refusal");
        const auto second = provider::provision(scope, data, cancel.get());
        expect(second.status == Status::ok && second.item.generation != item.generation, "distinct replacement generation");
        expect(provider::retire(item, cancel.get()) == Status::ok, "retire first generation");
        expect(use(item, 32) == Status::not_found && calls == 1, "retired item refusal");
        expect(use(second.item, 32) == Status::ok && calls == 2, "preserve other generation");
        expect(provider::retire(second.item, cancel.get()) == Status::ok, "retire second generation");
        g_cancellable_cancel(cancel.get());
        expect(use(second.item, 32) == Status::cancelled && calls == 2, "cancellation");
        return 0;
    } catch (const std::exception& e) {
        std::cerr << "Qualification failed: " << e.what() << '\n';
        return 1;
    }
}
