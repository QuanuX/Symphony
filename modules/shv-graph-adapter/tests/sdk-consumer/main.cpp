#include <symphony/graph/adapter.hpp>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <iostream>
#include <stdexcept>

namespace engine = symphony::knowledge::engine;
using symphony::graph::Json;
Json seal(Json value) {
    value["digest"] = engine::tagged_sha256(value.dump());
    return value;
}
int main() {
    static_assert(__cplusplus >= 202400L, "SDK must propagate C++26");
    const auto artifact = seal(Json{{"protocol", "example.sdk.subject.v1"}, {"purpose", "independent SDK fixture"}});
    auto graph = seal(Json{
        {"protocol", symphony::graph::exchange_protocol},
        {"owner", {{"engine_id", "example-owner"}, {"engine_version", "1"},
                   {"artifact_protocol", artifact.at("protocol")}, {"artifact_digest", artifact.at("digest")}}},
        {"owner_artifact", artifact},
        {"nodes", Json::array({
            Json{{"id", "a"}, {"labels", Json::array({"custom"})}, {"properties", {{"value", 11}}}},
            Json{{"id", "b"}, {"labels", Json::array({"custom"})}, {"properties", {{"value", "retained"}}}}
        })},
        {"edges", Json::array({Json{{"id", "a-to-b"}, {"from", "a"}, {"to", "b"},
                                   {"label", "custom-link"}, {"properties", Json::object()}}})}
    });
    symphony::graph::validate_exchange(graph);
    const symphony::graph::PortableReference reference;
    const symphony::graph::Adapter& port = reference;
    const auto roundtrip = port.roundtrip(graph);
    if (roundtrip.at("graph") != graph) throw std::runtime_error("roundtrip lost values");
    const auto query = port.query(graph, "nodes", {"b", "missing", "a"});
    if (query.at("rows").size() != 2 || query.at("rows").at(0).at("id") != "a" ||
        query.at("rows").at(1).at("id") != "b" || query.at("missing_ids") != Json::array({"missing"}))
        throw std::runtime_error("query correspondence failed");
    graph["edges"][0]["to"] = "missing";
    graph.erase("digest");
    graph = seal(graph);
    bool rejected = false;
    try { symphony::graph::validate_exchange(graph); }
    catch (const engine::Error&) { rejected = true; }
    if (!rejected) throw std::runtime_error("invalid endpoint was admitted");
    std::cout << "installed SDK: linkable validator, virtual port, exact roundtrip, ordered query, missing IDs, dangling endpoint rejection; C++ " << __cplusplus << '\n';
}
