#include "shv.hpp"
#include <algorithm>
#include <map>
namespace symphony::knowledge::shv {
namespace {
void ids(const Json &v) {
  array(v, 128);
  std::set<std::string> seen;
  for (const auto &id : v) {
    Json j = {{"id", id}};
    if (!seen.insert(ident(j, "id")).second)
      invalid("duplicate selected id");
  }
}
void require_value(const Json &v) {
  if (v.is_string()) {
    Json j = {{"value", v}};
    text(j, "value", 4096);
    return;
  }
  if (v.is_number_integer() || v.is_number_unsigned()) {
    if (v < -9007199254740991LL || v > 9007199254740991LL)
      invalid("integer outside interoperable range");
    return;
  }
  if (v.is_array()) {
    array(v, 32);
    std::set<std::string> seen;
    for (const auto &x : v) {
      Json j = {{"value", x}};
      if (!seen.insert(text(j, "value", 128)).second)
        invalid("duplicate requirement token");
    }
    return;
  }
  invalid("unsupported requirement value");
}
std::pair<Json, Json> selected(const Json &c, const Json &requested) {
  ids(requested);
  Json rows = Json::array(), missing = Json::array();
  std::map<std::string, Json> available;
  for (const auto &s : c.at("subjects"))
    available.emplace(s.at("id").get<std::string>(), s);
  if (requested.empty())
    return {c.at("subjects"), missing};
  for (const auto &id : requested) {
    auto it = available.find(id.get<std::string>());
    if (it == available.end())
      missing.push_back(id);
    else
      rows.push_back(it->second);
  }
  return {rows, missing};
}
} // namespace
Json handle_request(const engine::Request &r) {
  deadline(r);
  const auto &p = r.payload;
  if (r.operation == "inspect") {
    fields(p, {});
    return descriptor();
  }
  if (r.operation == "coverage_default")
    return coverage_default(p);
  if (r.operation == "coverage_plan")
    return coverage_plan(p);
  if (r.operation == "catalogue_build")
    return catalogue_build(r, p);
  if (r.operation == "graph_validate") {
    fields(p, {"source_root", "graph"});
    const auto &g = p.at("graph");
    fields(g,
           {"protocol", "owner", "owner_artifact", "nodes", "edges", "digest"});
    check_seal(g);
    catalogue_replay(r, g.at("owner_artifact"), text(p, "source_root"));
    if (project(g.at("owner_artifact")) != g)
      invalid("graph is not exact SHV projection");
    return seal(Json{{"protocol", "symphony.shv.graph-validation.v1"},
                     {"graph_digest", g.at("digest")},
                     {"catalogue_digest", g.at("owner_artifact").at("digest")},
                     {"valid", true}});
  }
  if (r.operation == "graph_project") {
    fields(p, {"source_root", "catalogue"});
    catalogue_replay(r, p.at("catalogue"), text(p, "source_root"));
    return project(p.at("catalogue"));
  }
  if (r.operation != "catalogue_query" && r.operation != "evaluate")
    invalid("unsupported operation");
  if (r.operation == "catalogue_query")
    fields(p, {"source_root", "catalogue", "subject_ids"});
  else
    fields(p, {"source_root", "catalogue", "subject_ids", "requirements"});
  const auto &c = p.at("catalogue");
  catalogue_replay(r, c, text(p, "source_root"));
  auto [rows, missing] = selected(c, p.at("subject_ids"));
  if (r.operation == "catalogue_query")
    return seal(Json{{"protocol", "symphony.shv.query-result.v1"},
                     {"catalogue_digest", c.at("digest")},
                     {"subject_ids", p.at("subject_ids")},
                     {"subjects", rows},
                     {"missing_subject_ids", missing}});
  array(p.at("requirements"), 32);
  std::set<std::string> reqids;
  for (const auto &req : p.at("requirements")) {
    fields(req, {"id", "predicate", "operator", "value", "qualifier"});
    if (!reqids.insert(ident(req, "id")).second)
      invalid("duplicate requirement");
    ident(req, "predicate");
    auto op = text(req, "operator", 16);
    text(req, "qualifier", 256);
    require_value(req.at("value"));
    if (op != "eq" && op != "gte" && op != "contains")
      invalid("unsupported comparator");
    if (op == "gte" && !req.at("value").is_number_integer() &&
        !req.at("value").is_number_unsigned())
      invalid("gte requires integer");
    if (op == "contains" && (!req.at("value").is_string() ||
                             req.at("qualifier") != "finite_supported_set"))
      invalid("contains requires token and finite_supported_set qualifier");
  }
  Json findings = Json::array();
  for (const auto &subject : rows) {
    deadline(r);
    for (const auto &req : p.at("requirements")) {
      std::string status = "unresolved";
      Json evidence = Json::array();
      for (const auto &a : subject.at("assertions")) {
        if (a.at("predicate") != req.at("predicate") ||
            a.at("qualifier") != req.at("qualifier"))
          continue;
        auto op = req.at("operator").get<std::string>();
        const auto &v = a.at("value");
        bool passes = false;
        if (op == "eq") {
          if (v.type() != req.at("value").type() &&
              !(v.is_number_integer() && req.at("value").is_number_integer()))
            invalid("incompatible eq value types");
          passes = v == req.at("value");
        } else if (op == "gte") {
          if (!v.is_number_integer() && !v.is_number_unsigned())
            invalid("incompatible gte source type");
          passes = v >= req.at("value");
        } else {
          if (!v.is_array())
            invalid("contains source is not finite token set");
          passes = std::find(v.begin(), v.end(), req.at("value")) != v.end();
        }
        evidence.push_back(a);
        status = passes ? "supported" : "contradicted";
      }
      findings.push_back(Json{{"subject_id", subject.at("id")},
                              {"requirement_id", req.at("id")},
                              {"status", status},
                              {"evidence", evidence}});
    }
  }
  return seal(Json{{"protocol", "symphony.shv.evaluation.v1"},
                   {"catalogue_digest", c.at("digest")},
                   {"subject_ids", p.at("subject_ids")},
                   {"requirements", p.at("requirements")},
                   {"findings", findings},
                   {"missing_subject_ids", missing}});
}
} // namespace symphony::knowledge::shv
