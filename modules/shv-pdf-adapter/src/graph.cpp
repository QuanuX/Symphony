#include "pdf.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include <algorithm>
namespace symphony::knowledge::shv_pdf {
Json project_graph(const Json &extraction) {
  const auto source_id =
      "source:" + extraction.at("source").at("digest").get<std::string>();
  const auto derivation_id =
      "derivation:" + extraction.at("digest").get<std::string>();
  Json nodes = Json::array(
      {Json{{"id", source_id},
            {"labels", Json::array({"DocumentSource"})},
            {"properties", extraction.at("source")}},
       Json{{"id", derivation_id},
            {"labels", Json::array({"DocumentDerivation"})},
            {"properties", Json{{"profile", extraction.at("profile")},
                                {"decoder", extraction.at("decoder")},
                                {"text_digest", extraction.at("text_digest")},
                                {"documentary_lineages", 1},
                                {"namespace_equivalence", "not_asserted"}}}}});
  Json edges = Json::array({Json{{"id", "derives:" + derivation_id},
                                 {"from", derivation_id},
                                 {"to", source_id},
                                 {"label", "derived_from"},
                                 {"properties", Json::object()}}});
  std::size_t index = 0;
  for (const auto &row : extraction.at("rows")) {
    const auto id = "row:" + extraction.at("digest").get<std::string>() + ":" +
                    row.at("opn").get<std::string>();
    nodes.push_back(Json{
        {"id", id},
        {"labels", Json::array({"DocumentSubject"})},
        {"properties", Json{{"model", row.at("model")},
                            {"source_id", extraction.at("source").at("id")},
                            {"identity_verified", false}}}});
    Json citation = {{"source_id", extraction.at("source").at("id")},
                     {"source_digest", extraction.at("source").at("digest")},
                     {"derivation_digest", extraction.at("digest")},
                     {"page_index", 12},
                     {"table", 8},
                     {"row_index", index++}};
    Json assertion = {{"predicate", "oem_identifier"},
                      {"qualifier", "issuer=AMD;namespace=opn;profile=1"},
                      {"value", row.at("opn")},
                      {"citation", citation}};
    edges.push_back(Json{{"id", "asserts:" + id},
                         {"from", source_id},
                         {"to", id},
                         {"label", "documents_identifier"},
                         {"properties", assertion}});
    edges.push_back(Json{{"id", "extracts:" + id},
                         {"from", derivation_id},
                         {"to", id},
                         {"label", "extracts_subject"},
                         {"properties", Json::object()}});
  }
  auto order = [](const Json &a, const Json &b) {
    return a.at("id") < b.at("id");
  };
  std::sort(nodes.begin(), nodes.end(), order);
  std::sort(edges.begin(), edges.end(), order);
  Json graph = {{"protocol", "symphony.graph.exchange.v1"},
                {"owner", Json{{"engine_id", engine_id},
                               {"engine_version", version},
                               {"artifact_protocol", extraction.at("protocol")},
                               {"artifact_digest", extraction.at("digest")}}},
                {"owner_artifact", extraction},
                {"nodes", nodes},
                {"edges", edges}};
  graph["digest"] = engine::tagged_sha256(graph.dump());
  return graph;
}
} // namespace symphony::knowledge::shv_pdf
