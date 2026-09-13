#pragma once
#include "symphony/knowledge/engine/json.hpp"
#include <string>
#include <vector>
namespace symphony::graph {
using Json = knowledge::engine::Json;
inline constexpr const char* exchange_protocol = "symphony.graph.exchange.v1";
// Structural conformance only; the referenced semantic owner validates meaning.
void validate_exchange(const Json& graph);
class Adapter {
public:
    virtual ~Adapter() = default;
    virtual Json roundtrip(const Json& graph) const = 0;
    virtual Json query(const Json& graph, const std::string& kind,
                       const std::vector<std::string>& ids) const = 0;
};
class PortableReference final : public Adapter {
public:
    Json roundtrip(const Json& graph) const override;
    Json query(const Json& graph, const std::string& kind,
               const std::vector<std::string>& ids) const override;
};
}
