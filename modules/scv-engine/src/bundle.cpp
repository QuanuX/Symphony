#include "bundle.hpp"
#include "composition.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <algorithm>
#include <map>
#include <set>
#include <string_view>
#include <vector>

namespace symphony::knowledge::scv {
namespace {
constexpr std::size_t logical_bytes = 4U << 20U;
constexpr std::size_t logical_values = 32768;
constexpr std::size_t logical_depth = 64;
constexpr std::size_t object_limit = 2048;
constexpr std::size_t edge_limit = 8192;
constexpr std::size_t step_limit = 16384;
constexpr std::size_t materialized_limit = 64U << 20U;
constexpr const char* bundle_protocol = "symphony.scv.evidence-bundle.v1";

[[noreturn]] void invalid(const std::string& message) {
    throw engine::Error("bundle.invalid", message, 4);
}
void deadline(const engine::Request& request) {
    if (engine::unix_time_ms() > request.deadline_unix_ms)
        throw engine::Error("request.deadline_exceeded", "bundle deadline exceeded", 4);
}
void fields(const Json& value, std::initializer_list<std::string_view> names) {
    if (!value.is_object() || value.size() != names.size()) invalid("unexpected bundle fields");
    for (const auto name : names) if (!value.contains(std::string(name))) invalid("missing bundle field");
}
std::size_t add(std::size_t left, std::size_t right, std::size_t maximum) {
    if (right > maximum || left > maximum - right) invalid("bundle expansion budget exceeded");
    return left + right;
}
std::string digest_text(const Json& value) {
    if (!value.is_string()) invalid("expected bundle digest");
    const auto& s = value.get_ref<const std::string&>();
    if (s.size() != 71 || !s.starts_with("sha256:") ||
        !std::all_of(s.begin() + 7, s.end(), [](char c) { return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'); }))
        invalid("invalid bundle digest");
    return s;
}
std::string canonical(const Json& value) {
    try { return value.dump(-1, ' ', false, Json::error_handler_t::strict); }
    catch (const Json::exception&) { invalid("bundle contains invalid UTF-8 JSON"); }
}
std::string full_digest(const Json& value, const engine::Request& request) {
    deadline(request);
    const auto result = engine::tagged_sha256(canonical(value));
    deadline(request);
    return result;
}
std::size_t scalar_bytes(const Json& value) {
    if (value.is_structured() || value.is_number_float() || value.is_discarded() || value.is_binary())
        invalid("bundle scalar must be null, boolean, bounded integer or string");
    if (value.is_string() && value.get_ref<const std::string&>().size() > engine::Limits::max_string_bytes)
        invalid("bundle string exceeds byte limit");
    if (value.is_number_unsigned()) {
        if (value.get<std::uint64_t>() > 9007199254740991ULL) invalid("bundle integer exceeds interoperable range");
    } else if (value.is_number_integer()) {
        const auto number = value.get<std::int64_t>();
        if (number < -9007199254740991LL || number > 9007199254740991LL)
            invalid("bundle integer exceeds interoperable range");
    }
    return canonical(value).size();
}
struct Shape {
    std::size_t bytes = 2;
    std::size_t values = 1;
    std::size_t depth = 0;
};
void member(Shape& shape, const Shape& child, bool first, const std::string* key) {
    if (!first) shape.bytes = add(shape.bytes, 1, logical_bytes);
    if (key) {
        shape.bytes = add(shape.bytes, add(scalar_bytes(Json(*key)), 1, logical_bytes), logical_bytes);
        shape.values = add(shape.values, 1, logical_values);
    }
    shape.bytes = add(shape.bytes, child.bytes, logical_bytes);
    shape.values = add(shape.values, child.values, logical_values);
    shape.depth = std::max(shape.depth, add(child.depth, 1, logical_depth));
}
Json metrics(const Shape& root, std::size_t objects, std::size_t refs, std::size_t materialized) {
    return {{"object_count", objects}, {"reference_count", refs}, {"traversal_steps", objects + refs},
        {"expanded_bytes", root.bytes}, {"expanded_values", root.values}, {"expanded_depth", root.depth},
        {"materialized_bytes", materialized}};
}

class Decoder {
public:
    Decoder(const Json& bundle, const engine::Request& request) : bundle_(bundle), request_(request) {}
    DecodedBundle run() {
        deadline(request_);
        fields(bundle_, {"protocol", "root_digest", "objects", "digest"});
        if (bundle_.at("protocol") != bundle_protocol) invalid("unsupported evidence bundle protocol");
        const auto seal = digest_text(bundle_.at("digest"));
        // The caller's ordinary process parser has already bounded the wire.
        // Direct library use receives the same structural/budget check here.
        static_cast<void>(engine::parse_bounded_json(canonical(bundle_), engine::Limits::max_response_bytes));
        if (sealed(bundle_).at("digest") != seal) invalid("bundle self-seal mismatch");
        const auto root = digest_text(bundle_.at("root_digest"));
        const auto& objects = bundle_.at("objects");
        if (!objects.is_array() || objects.empty() || objects.size() > object_limit) invalid("bundle object count outside bounds");
        std::string previous;
        for (const auto& node : objects) {
            deadline(request_);
            if (!node.is_object() || !node.contains("kind")) invalid("invalid bundle node");
            if (node.at("kind") == "object") fields(node, {"digest", "kind", "members"});
            else if (node.at("kind") == "array") fields(node, {"digest", "kind", "items"});
            else invalid("unknown bundle node kind");
            const auto id = digest_text(node.at("digest"));
            if (!previous.empty() && id <= previous) invalid("bundle objects must have unique sorted digests");
            previous = id;
            const bool object = node.at("kind") == "object";
            const auto& children = node.at(object ? "members" : "items");
            if (object ? !children.is_object() : !children.is_array()) invalid("bundle node has wrong container type");
            for (auto it = children.begin(); it != children.end(); ++it) {
                if (object) static_cast<void>(scalar_bytes(Json(it.key())));
                const auto& child = it.value();
                if (!child.is_object() || child.size() != 1) invalid("invalid bundle child");
                if (child.contains("scalar")) static_cast<void>(scalar_bytes(child.at("scalar")));
                else if (child.contains("ref")) {
                    static_cast<void>(digest_text(child.at("ref")));
                    references_ = add(references_, 1, edge_limit);
                } else invalid("unknown bundle child kind");
            }
            nodes_.emplace(id, &node);
        }
        static_cast<void>(add(nodes_.size(), references_, step_limit));
        if (!nodes_.contains(root) || nodes_.at(root)->at("kind") != "object") invalid("bundle root must resolve to an object");
        const auto shape = measure(root, 0);
        if (shapes_.size() != nodes_.size()) invalid("bundle contains unreachable objects");
        // All expansion and cumulative work limits are established before any
        // expanded node is allocated. Repeated child occurrences were counted.
        std::map<std::string, Json> expanded;
        for (const auto& id : order_) {
            deadline(request_);
            const auto& node = *nodes_.at(id);
            const bool object = node.at("kind") == "object";
            Json value = object ? Json::object() : Json::array();
            const auto& children = node.at(object ? "members" : "items");
            for (auto it = children.begin(); it != children.end(); ++it) {
                deadline(request_);
                const auto& child = it.value();
                const auto& item = child.contains("scalar") ? child.at("scalar") : expanded.at(child.at("ref").get<std::string>());
                if (object) value[it.key()] = item;
                else value.push_back(item);
            }
            if (full_digest(value, request_) != id) invalid("bundle expanded node digest mismatch");
            expanded.emplace(id, std::move(value));
        }
        auto value = std::move(expanded.at(root));
        return {std::move(value), metrics(shape, nodes_.size(), references_, materialized_)};
    }
private:
    Shape measure(const std::string& id, std::size_t depth) {
        deadline(request_);
        if (depth > logical_depth) invalid("bundle expansion depth exceeded");
        if (!nodes_.contains(id)) invalid("bundle references a missing object");
        if (active_.contains(id)) invalid("bundle contains a reference cycle");
        if (shapes_.contains(id)) return shapes_.at(id);
        active_.insert(id);
        const auto& node = *nodes_.at(id);
        const bool object = node.at("kind") == "object";
        const auto& children = node.at(object ? "members" : "items");
        Shape shape;
        bool first = true;
        for (auto it = children.begin(); it != children.end(); ++it) {
            deadline(request_);
            const auto& child = it.value();
            const auto child_shape = child.contains("scalar") ? Shape{scalar_bytes(child.at("scalar")), 1, 0} :
                measure(child.at("ref").get<std::string>(), depth + 1);
            if (object) { const auto key = it.key(); member(shape, child_shape, first, &key); }
            else member(shape, child_shape, first, nullptr);
            first = false;
        }
        materialized_ = add(materialized_, shape.bytes, materialized_limit);
        active_.erase(id);
        shapes_.emplace(id, shape);
        order_.push_back(id);
        return shape;
    }
    const Json& bundle_;
    const engine::Request& request_;
    std::map<std::string, const Json*> nodes_;
    std::map<std::string, Shape> shapes_;
    std::set<std::string> active_;
    std::vector<std::string> order_;
    std::size_t references_ = 0;
    std::size_t materialized_ = 0;
};

class Encoder {
public:
    explicit Encoder(const engine::Request& request) : request_(request) {}
    Json run(const Json& value) {
        if (!value.is_object()) invalid("bundle root must be an object");
        const auto root = visit(value, 0).first;
        Json objects = Json::array();
        for (const auto& [id, node] : nodes_) { static_cast<void>(id); objects.push_back(node); }
        auto result = sealed(Json{{"protocol", bundle_protocol}, {"root_digest", root}, {"objects", objects}});
        static_cast<void>(engine::parse_bounded_json(canonical(result), engine::Limits::max_response_bytes));
        return result;
    }
    Json result_metrics() const { return metrics(root_shape_, nodes_.size(), references_, materialized_); }
private:
    std::pair<std::string, Shape> visit(const Json& value, std::size_t depth) {
        deadline(request_);
        if (depth > logical_depth) invalid("bundle expansion depth exceeded");
        const bool object = value.is_object();
        if (!object && !value.is_array()) invalid("bundle reference must name a container");
        Json children = object ? Json::object() : Json::array();
        Shape shape;
        std::size_t refs = 0;
        bool first = true;
        for (auto it = value.begin(); it != value.end(); ++it) {
            deadline(request_);
            Json child;
            Shape child_shape;
            if (it.value().is_structured()) {
                const auto encoded = visit(it.value(), depth + 1);
                child = {{"ref", encoded.first}};
                child_shape = encoded.second;
                refs = add(refs, 1, edge_limit);
            } else {
                child_shape = {scalar_bytes(it.value()), 1, 0};
                child = {{"scalar", it.value()}};
            }
            if (object) { const auto key = it.key(); member(shape, child_shape, first, &key); children[key] = std::move(child); }
            else { member(shape, child_shape, first, nullptr); children.push_back(std::move(child)); }
            first = false;
        }
        const auto id = full_digest(value, request_);
        if (!nodes_.contains(id)) {
            static_cast<void>(add(nodes_.size(), 1, object_limit));
            references_ = add(references_, refs, edge_limit);
            static_cast<void>(add(nodes_.size() + 1, references_, step_limit));
            materialized_ = add(materialized_, shape.bytes, materialized_limit);
            nodes_.emplace(id, Json{{"digest", id}, {"kind", object ? "object" : "array"}, {object ? "members" : "items", std::move(children)}});
        }
        if (depth == 0) root_shape_ = shape;
        return {id, shape};
    }
    const engine::Request& request_;
    std::map<std::string, Json> nodes_;
    std::size_t references_ = 0;
    std::size_t materialized_ = 0;
    Shape root_shape_;
};

std::string invocation(const Json& input, const std::string& domain) {
    fields(input, {"operation", "owner", "bundle"});
    fields(input.at("owner"), {"domain", "version"});
    if (input.at("owner").at("domain") != domain || input.at("owner").at("version") != version)
        invalid("bundle evaluator identity differs from invoked owner");
    if (!input.at("operation").is_string()) invalid("invalid bundled logical operation");
    const auto op = input.at("operation").get<std::string>();
    if (op != "composition_explore" && op != "composition_reassess" && op != "composition_obligations" && op != "composition_followup")
        invalid("bundled logical operation is not allowlisted");
    return op;
}
}

Json encode_bundle(const Json& value, const engine::Request& request) { return Encoder(request).run(value); }
DecodedBundle decode_bundle(const Json& bundle, const engine::Request& request) { return Decoder(bundle, request).run(); }
Json handle_bundle(const engine::Request& request, const std::string& domain) {
    deadline(request);
    const auto op = invocation(request.payload, domain);
    auto decoded = decode_bundle(request.payload.at("bundle"), request);
    const auto input_digest = full_digest(request.payload, request);
    if (request.operation == "bundle_inspect")
        return sealed(Json{{"protocol", "symphony.scv.bundle-inspection.v1"}, {"domain", domain}, {"operation", op},
            {"owner", request.payload.at("owner")}, {"input_digest", input_digest}, {"root_digest", request.payload.at("bundle").at("root_digest")},
            {"metrics", decoded.metrics}, {"validation", "transport_only"}, {"limitations", Json::array({
                "bundle validation establishes exact bounded reconstruction only; it does not validate composition semantics or authenticate evidence producers",
                "all dependencies are explicit and complete; no ambient source, graph, filesystem or network lookup occurs"})}});
    if (request.operation != "composition_bundle_evaluate") invalid("unsupported bundle operation");
    auto logical = request;
    logical.operation = op;
    logical.payload = std::move(decoded.value);
    const auto result = handle_composition(logical, domain);
    Encoder encoder(request);
    auto result_bundle = encoder.run(result);
    const auto result_metrics = encoder.result_metrics();
    deadline(request);
    return sealed(Json{{"protocol", "symphony.scv.composition-bundle-evaluation.v1"}, {"domain", domain}, {"operation", op},
        {"owner", request.payload.at("owner")}, {"input_digest", input_digest}, {"input_root_digest", request.payload.at("bundle").at("root_digest")},
        {"input_metrics", decoded.metrics}, {"result_bundle", std::move(result_bundle)}, {"native_result_digest", result.at("digest")},
        {"result_metrics", result_metrics}, {"validation", "owner_evaluated"}, {"limitations", Json::array({
            "bundled execution preserves the existing composition owner's meanings and caller criteria; it does not certify deployment or authenticate submitted provenance",
            "content references reduce repeated transport; expanded values and graph work remain explicitly bounded",
            "historical installation identity belongs to retained owner records; a transport content digest is not a receipt"})}});
}
}
