#include "profile.hpp"
#include "shv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <algorithm>
#include <deque>
#include <filesystem>
#include <map>
#include <regex>
#include <set>
namespace symphony::knowledge::shv_profile {
namespace k = symphony::knowledge::shv;
namespace {
Json failure(const engine::Error &e) {
  if (e.code() == "request.deadline")
    throw e;
  return Json{{"code", e.code()}, {"message", e.what()}};
}
Json attempt(const engine::Request &r, const Json &p, Json &error) {
  k::deadline(r);
  try {
    return k::catalogue_build(r, p);
  } catch (const engine::Error &e) {
    error = failure(e);
    return nullptr;
  }
}
void digest(const Json &v) {
  static const std::regex pattern("sha256:[0-9a-f]{64}");
  if (!v.is_string() || !std::regex_match(v.get<std::string>(), pattern))
    k::invalid("reference digest required");
}
} // namespace
Json extraction_diagnose(const engine::Request &r) {
  const auto &p = r.payload;
  k::fields(p, {"source_root", "sources", "subjects"});
  auto root = k::text(p, "source_root");
  if (std::any_of(root.begin(), root.end(),
                  [](unsigned char c) { return c < 32 || c == 127; }) ||
      !std::filesystem::path(root).is_absolute() ||
      std::filesystem::path(root).lexically_normal().string() != root ||
      (root.size() > 1 && root.back() == '/'))
    k::invalid("clean absolute source root required");
  validate_sources(p.at("sources"));
  validate_mappings(p.at("subjects"), false);
  std::map<std::string, Json> manifests;
  std::set<std::string> verified;
  for (const auto &s : p.at("sources"))
    manifests.emplace(k::ident(s, "id"), s);
  for (const auto &m : p.at("subjects")) {
    if (!manifests.contains(k::ident(m, "source_id")))
      k::invalid("mapping source absent");
    if (m.contains("document") && m.at("document").contains("decoder_binding"))
      k::invalid("diagnosis requires bound decoder roots");
  }
  Json sources = Json::array(), subjects = Json::array();
  Json counts = {{"extracted", 0}, {"failed", 0}, {"unavailable", 0}};
  for (const auto &s : p.at("sources")) {
    Json error = nullptr;
    auto result = attempt(r,
                          Json{{"source_root", root},
                               {"sources", Json::array({s})},
                               {"subjects", Json::array()}},
                          error);
    auto id = k::ident(s, "id");
    if (!result.is_null())
      verified.insert(id);
    sources.push_back(
        Json{{"source_id", id},
             {"status", result.is_null() ? "unverified" : "verified"},
             {"error", error}});
  }
  for (const auto &m : p.at("subjects")) {
    Json fields = Json::array();
    auto source = k::ident(m, "source_id");
    for (const auto &f : m.at("fields")) {
      Json error = nullptr, assertion = nullptr;
      std::string status = "unavailable";
      if (verified.contains(source)) {
        auto single = m;
        single["fields"] = Json::array({f});
        auto c = attempt(r,
                         Json{{"source_root", root},
                              {"sources", Json::array({manifests.at(source)})},
                              {"subjects", Json::array({single})}},
                         error);
        status = c.is_null() ? "failed" : "extracted";
        if (!c.is_null())
          assertion = c.at("subjects").at(0).at("assertions").at(0);
      }
      counts[status] = counts[status].get<int>() + 1;
      fields.push_back(Json{{"predicate", f.at("predicate")},
                            {"status", status},
                            {"assertion", assertion},
                            {"error", error}});
    }
    subjects.push_back(Json{
        {"subject_id", m.at("id")}, {"source_id", source}, {"fields", fields}});
  }
  // Prevent successful fields from spanning a source replacement during the
  // run.
  for (const auto &id : verified)
    static_cast<void>(
        k::catalogue_build(r, Json{{"source_root", root},
                                   {"sources", Json::array({manifests.at(id)})},
                                   {"subjects", Json::array()}}));
  return k::seal(Json{{"protocol", "symphony.shv.extraction-diagnostics.v1"},
                      {"input", p},
                      {"reader",
                       {{"engine_id", k::engine_id},
                        {"version", k::version},
                        {"mode", "compiled_exact_contract"}}},
                      {"sources", sources},
                      {"subjects", subjects},
                      {"counts", counts},
                      {"evidence_scope", "individual_field_source_replay"},
                      {"canonical_apply_enabled", false}});
}
Json references_analyze(const Json &p) {
  k::fields(p, {"objects", "edges", "root_ids", "candidate_ids"});
  k::array(p.at("objects"), 128);
  k::array(p.at("edges"), 512);
  std::map<std::string, Json> objects;
  Json uninspected = Json::array();
  for (const auto &o : p.at("objects")) {
    k::fields(o, {"id", "kind", "digest", "document"});
    auto id = k::ident(o, "id");
    k::ident(o, "kind");
    digest(o.at("digest"));
    if (!objects.emplace(id, o).second)
      k::invalid("duplicate reference object");
    const auto &d = o.at("document");
    if (!d.is_null()) {
      if (!d.is_object())
        k::invalid("reference document must be an object");
      if (d.contains("digest")) {
        k::check_seal(d);
        if (d.at("digest") != o.at("digest"))
          k::invalid("sealed document identity differs");
      } else if (engine::tagged_sha256(d.dump()) !=
                 o.at("digest").get<std::string>())
        k::invalid("canonical document identity differs");
    }
  }
  for (const auto &[id, o] : objects)
    if (o.at("document").is_null())
      uninspected.push_back(id);
  auto ids = [&](const Json &v) {
    k::array(v, 128);
    std::set<std::string> out;
    for (const auto &id : v) {
      auto s = k::ident(Json{{"id", id}}, "id");
      if (!objects.contains(s) || !out.insert(s).second)
        k::invalid("unknown or repeated selected object");
    }
    return out;
  };
  auto roots = ids(p.at("root_ids"));
  static_cast<void>(ids(p.at("candidate_ids")));
  std::map<std::string, std::set<std::string>> adjacency;
  std::set<std::tuple<std::string, std::string, std::string>> seen;
  Json edges = Json::array();
  for (const auto &e : p.at("edges")) {
    k::fields(e, {"from", "to", "pointer"});
    auto from = k::ident(e, "from"), to = k::ident(e, "to"),
         pointer = k::text(e, "pointer", 4096);
    if (std::any_of(pointer.begin(), pointer.end(),
                    [](unsigned char c) { return c < 32 || c == 127; }))
      k::invalid("control character in reference pointer");
    if (!objects.contains(from) || !objects.contains(to) ||
        objects.at(from).at("document").is_null() ||
        !seen.emplace(from, pointer, to).second)
      k::invalid("unknown, uninspected or repeated reference edge");
    try {
      if (objects.at(from).at("document").at(Json::json_pointer(pointer)) !=
          objects.at(to).at("digest"))
        k::invalid("reference pointer does not match target digest");
    } catch (const Json::exception &) {
      k::invalid("invalid or unresolved reference pointer");
    }
    adjacency[from].insert(to);
  }
  for (const auto &[from, pointer, to] : seen)
    edges.push_back(Json{{"from", from}, {"to", to}, {"pointer", pointer}});
  std::map<std::string, Json> paths;
  std::deque<std::string> queue;
  for (const auto &id : roots) {
    paths[id] = Json::array({id});
    queue.push_back(id);
  }
  while (!queue.empty()) {
    auto id = queue.front();
    queue.pop_front();
    for (const auto &to : adjacency[id])
      if (!paths.contains(to)) {
        paths[to] = paths.at(id);
        paths[to].push_back(to);
        queue.push_back(to);
      }
  }
  Json reachable = Json::array(), candidates = Json::array();
  for (const auto &[id, path] : paths) {
    static_cast<void>(path);
    reachable.push_back(id);
  }
  for (const auto &v : p.at("candidate_ids")) {
    auto id = v.get<std::string>();
    Json incoming = Json::array();
    for (const auto &e : edges)
      if (e.at("to") == id)
        incoming.push_back(e);
    candidates.push_back(
        Json{{"id", id},
             {"status", paths.contains(id) ? "reachable"
                                           : "not_reachable_in_supplied_graph"},
             {"path", paths.contains(id) ? paths.at(id) : Json(nullptr)},
             {"incoming_edges", incoming}});
  }
  return k::seal(Json{{"protocol", "symphony.shv.reference-analysis.v1"},
                      {"input", p},
                      {"edges", edges},
                      {"reachable_ids", reachable},
                      {"candidates", candidates},
                      {"uninspected_object_ids", uninspected},
                      {"evidence_scope", "caller_selected_reference_edges"},
                      {"deletion_authorized", false},
                      {"canonical_apply_enabled", false}});
}
} // namespace symphony::knowledge::shv_profile
