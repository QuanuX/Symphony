#include "symphony/ssiag/linux_secret_service.hpp"

#include <libsecret/secret.h>
#include <unistd.h>
#include <algorithm>
#include <memory>
#include <string_view>

namespace {
// Secret operations may return a prompt if the screen locks mid-request. Never
// run that prompt. Refusal is safe in a noninteractive owner invocation.
struct QuietService { SecretService parent; };
struct QuietServiceClass { SecretServiceClass parent; };
G_DEFINE_TYPE(QuietService, quiet_service, SECRET_TYPE_SERVICE)
void quiet_service_init(QuietService*) {}
void quiet_service_class_init(QuietServiceClass* cls) {
    // libsecret 0.21.x's base constructor overwrites g-name with an
    // environment-derived default, even when a bus name was passed to open.
    // Restore the unique name before GInitable connects or negotiates a session.
    G_OBJECT_CLASS(cls)->constructor = [](GType type, guint count, GObjectConstructParam* params) -> GObject* {
        const char* pinned = nullptr;
        for (guint i = 0; i < count; ++i) {
            if (g_str_equal(params[i].pspec->name, "g-name")) pinned = g_value_get_string(params[i].value);
        }
        auto* obj = G_OBJECT_CLASS(quiet_service_parent_class)->constructor(type, count, params);
        // The parent also forces G_BUS_TYPE_SESSION. Keep the dedicated
        // g-connection supplied at construction instead of GLib's shared bus.
        g_object_set(obj, "g-name", pinned, "g-bus-type", G_BUS_TYPE_NONE,
                     "g-flags", G_DBUS_PROXY_FLAGS_DO_NOT_AUTO_START, nullptr);
        return obj;
    };
    auto* base = SECRET_SERVICE_CLASS(cls);
    base->prompt_sync = [](SecretService*, SecretPrompt*, GCancellable*, const GVariantType*, GError** error) -> GVariant* {
        g_set_error_literal(error, SECRET_ERROR, SECRET_ERROR_IS_LOCKED, "Interaction disabled");
        return nullptr;
    };
    base->prompt_async = [](SecretService* service, SecretPrompt*, const GVariantType*, GCancellable* cancel,
                            GAsyncReadyCallback callback, gpointer data) {
        auto* task = g_task_new(service, cancel, callback, data);
        g_task_return_new_error(task, SECRET_ERROR, SECRET_ERROR_IS_LOCKED, "Interaction disabled");
        g_object_unref(task);
    };
    base->prompt_finish = [](SecretService*, GAsyncResult* result, GError** error) -> GVariant* {
        return static_cast<GVariant*>(g_task_propagate_pointer(G_TASK(result), error));
    };
}
struct Unref { void operator()(gpointer ptr) const { g_object_unref(ptr); } };
template<class T> using Object = std::unique_ptr<T, Unref>;
using Variant = std::unique_ptr<GVariant, decltype(&g_variant_unref)>;
using Attrs = std::unique_ptr<GHashTable, decltype(&g_hash_table_unref)>;
using Value = std::unique_ptr<SecretValue, decltype(&secret_value_unref)>;
void release_list(GList* list) { g_list_free_full(list, g_object_unref); }
using Objects = std::unique_ptr<GList, decltype(&release_list)>;
struct Error { GError* value = nullptr; ~Error() { g_clear_error(&value); } };
}

namespace symphony::ssiag::linux_secret_service {
namespace {
constexpr const char* service_name = "org.freedesktop.secrets";
const SecretSchema schema = [] {
    SecretSchema s{};
    s.name = "org.quanux.Symphony.SSIAG.GenericCredential.v1";
    s.flags = SECRET_SCHEMA_NONE;
    s.attributes[0] = {"tops", SECRET_SCHEMA_ATTRIBUTE_STRING};
    s.attributes[1] = {"reference", SECRET_SCHEMA_ATTRIBUTE_STRING};
    s.attributes[2] = {"generation", SECRET_SCHEMA_ATTRIBUTE_STRING};
    return s;
}();

bool uuid(const std::string& s) {
    return s.size() == 36 && g_uuid_string_is_valid(s.c_str()) &&
           std::none_of(s.begin(), s.end(), [](char ch) { return ch >= 'A' && ch <= 'F'; });
}
bool valid(const Scope& s) {
    constexpr std::string_view prefix = "/org/freedesktop/secrets/collection/";
    return uuid(s.tops_id) && uuid(s.reference) && s.collection_path.size() <= 512 &&
           s.collection_path.find('\0') == std::string::npos && s.collection_path.starts_with(prefix) &&
           s.collection_path.find('/', prefix.size()) == std::string::npos && g_variant_is_object_path(s.collection_path.c_str());
}
Attrs attributes(const Item& item) {
    return {secret_attributes_build(&schema, "tops", item.scope.tops_id.c_str(), "reference", item.scope.reference.c_str(),
                                    "generation", item.generation.c_str(), nullptr), g_hash_table_unref};
}
Status failure(const Error& error, bool mutation = false) {
    if (g_error_matches(error.value, SECRET_ERROR, SECRET_ERROR_IS_LOCKED)) return Status::locked;
    if (mutation) return Status::indeterminate; // A lost response does not prove absence of an effect.
    if (g_error_matches(error.value, G_IO_ERROR, G_IO_ERROR_CANCELLED)) return Status::cancelled;
    return Status::unavailable;
}
// Require an explicit local session address. GLib's implicit autolaunch may
// start an unrelated desktop bus when invoked from SSH or a system service.
bool explicit_bus(const std::string& address) {
    return getuid() == geteuid() && geteuid() != 0 && address.starts_with("unix:path=/") &&
           address.find(';') == address.npos && g_dbus_is_address(address.c_str());
}
struct Session {
    Object<SecretService> service;
    Object<SecretCollection> collection;
    Status open(const Scope& scope, GCancellable* cancel) {
        if (!valid(scope) || !cancel) return Status::invalid;
        if (g_cancellable_is_cancelled(cancel)) return Status::cancelled;
        const char* selected = g_getenv("DBUS_SESSION_BUS_ADDRESS");
        const std::string address = selected ? selected : "";
        if (!explicit_bus(address)) return Status::unavailable;
        Error error;
        Object<GDBusConnection> bus(g_dbus_connection_new_for_address_sync(address.c_str(),
            static_cast<GDBusConnectionFlags>(G_DBUS_CONNECTION_FLAGS_AUTHENTICATION_CLIENT |
                                              G_DBUS_CONNECTION_FLAGS_MESSAGE_BUS_CONNECTION),
            nullptr, cancel, &error.value));
        if (!bus) return failure(error);
        auto bus_query = [&](const char* method, const char* name, const GVariantType* type) {
            return Variant(g_dbus_connection_call_sync(bus.get(), "org.freedesktop.DBus", "/org/freedesktop/DBus",
                      "org.freedesktop.DBus", method, g_variant_new("(s)", name), type, G_DBUS_CALL_FLAGS_NO_AUTO_START,
                      5000, cancel, &error.value), g_variant_unref);
        };
        auto owner_reply = bus_query("GetNameOwner", service_name, G_VARIANT_TYPE("(s)"));
        if (!owner_reply) return failure(error);
        const char* owner = nullptr;
        g_variant_get(owner_reply.get(), "(&s)", &owner);
        auto uid_reply = bus_query("GetConnectionUnixUser", owner, G_VARIANT_TYPE("(u)"));
        if (!uid_reply) return failure(error);
        guint32 uid;
        g_variant_get(uid_reply.get(), "(u)", &uid);
        if (uid != geteuid()) return Status::unavailable;
        service.reset(SECRET_SERVICE(g_initable_new(quiet_service_get_type(), cancel, &error.value,
            "g-connection", bus.get(), "g-name", owner, "flags", SECRET_SERVICE_OPEN_SESSION | SECRET_SERVICE_LOAD_COLLECTIONS, nullptr)));
        if (!service) return failure(error);
        auto* proxy = G_DBUS_PROXY(service.get());
        if (g_dbus_proxy_get_connection(proxy) != bus.get() || g_strcmp0(g_dbus_proxy_get_name(proxy), owner) != 0)
            return Status::unavailable;
        if (g_strcmp0(secret_service_get_session_algorithms(service.get()), "dh-ietf1024-sha256-aes128-cbc-pkcs7") != 0)
            return Status::unavailable;
        Objects all(secret_service_get_collections(service.get()), release_list);
        for (auto* link = all.get(); link != nullptr; link = link->next) {
            auto* candidate = SECRET_COLLECTION(link->data);
            if (scope.collection_path == g_dbus_proxy_get_object_path(G_DBUS_PROXY(candidate))) {
                collection.reset(SECRET_COLLECTION(g_object_ref(candidate)));
                return secret_collection_get_locked(candidate) ? Status::locked : Status::ok;
            }
        }
        return Status::unavailable;
    }
    Status find(const Item& item, GCancellable* cancel, Object<SecretItem>& result) {
        if (!uuid(item.generation)) return Status::invalid;
        auto status = open(item.scope, cancel);
        if (status != Status::ok) return status;
        auto attrs = attributes(item);
        Error error;
        Objects hits(secret_collection_search_sync(collection.get(), &schema, attrs.get(), SECRET_SEARCH_ALL, cancel, &error.value), release_list);
        if (error.value) return failure(error);
        if (!hits) return Status::not_found;
        if (hits->next) return Status::ambiguous;
        auto* matched = SECRET_ITEM(hits->data);
        if (secret_item_get_locked(matched)) return Status::locked;
        result.reset(SECRET_ITEM(g_object_ref(matched)));
        return Status::ok;
    }
};
}

ProvisionResult provision(const Scope& scope, Bytes data, GCancellable* cancel) {
    ProvisionResult result{Status::invalid, {scope, {}}};
    if (data.empty() || data.size() > maximum_size) return result;
    Session session;
    result.status = session.open(scope, cancel);
    if (result.status != Status::ok) return result;
    std::unique_ptr<gchar, decltype(&g_free)> generation(g_uuid_string_random(), g_free);
    result.item.generation = generation.get();
    auto attrs = attributes(result.item);
    Value value(secret_value_new(reinterpret_cast<const gchar*>(data.data()), data.size(), "application/octet-stream"), secret_value_unref);
    Error error;
    Object<SecretItem> created(secret_item_create_sync(session.collection.get(), &schema, attrs.get(), "Symphony credential",
                                        value.get(), SECRET_ITEM_CREATE_NONE, cancel, &error.value));
    result.status = created ? Status::ok : failure(error, true);
    return result;
}

Status use(const Item& item, std::size_t byte_ceiling, GCancellable* cancel, const Consumer& consumer) {
    if (byte_ceiling == 0 || byte_ceiling > maximum_size || !consumer) return Status::invalid;
    Session session;
    Object<SecretItem> match;
    const auto status = session.find(item, cancel, match);
    if (status != Status::ok) return status;
    Error error;
    if (!secret_item_load_secret_sync(match.get(), cancel, &error.value)) return failure(error);
    Value value(secret_item_get_secret(match.get()), secret_value_unref);
    // GNOME Keyring can return text/plain even for opaque binary items.
    // Use the length returned with the bytes, never string length or MIME hints.
    if (!value) return Status::unavailable;
    gsize length{};
    const auto* bytes = secret_value_get(value.get(), &length);
    if (!bytes || length == 0 || length > byte_ceiling) return Status::invalid;
    if (g_cancellable_is_cancelled(cancel)) return Status::cancelled;
    try { consumer(Bytes(reinterpret_cast<const std::byte*>(bytes), length)); }
    catch (...) { return Status::indeterminate; }
    return Status::ok;
}

Status retire(const Item& item, GCancellable* cancel) {
    Session session;
    Object<SecretItem> match;
    const auto status = session.find(item, cancel, match);
    if (status != Status::ok) return status;
    Error error;
    return secret_item_delete_sync(match.get(), cancel, &error.value) ? Status::ok : failure(error, true);
}
}
