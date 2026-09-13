#include "source.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include <algorithm>
#include <charconv>
#include <chrono>
#include <filesystem>
#include <map>
#include <set>
namespace symphony::knowledge::shv_source {
namespace {
[[noreturn]] void invalid(const std::string &why) {
  throw engine::Error("shv-source.invalid", why, 4);
}
void fields(const Json &j, std::initializer_list<const char *> keys) {
  if (!j.is_object() || j.size() != keys.size())
    invalid("unexpected object fields");
  for (auto k : keys)
    if (!j.contains(k))
      invalid(std::string("missing field: ") + k);
}
std::string text(const Json &j, const char *key, std::size_t max = 4096) {
  if (!j.contains(key) || !j.at(key).is_string())
    invalid("expected text");
  auto s = j.at(key).get<std::string>();
  if (s.empty() || s.size() > max ||
      std::any_of(s.begin(), s.end(),
                  [](unsigned char c) { return c < 32 || c == 127; }))
    invalid("text outside bounds");
  return s;
}
std::string id(const Json &j, const char *key) {
  auto s = text(j, key, 128);
  if (!std::all_of(s.begin(), s.end(), [](unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
               (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-';
      }))
    invalid("invalid identifier");
  return s;
}
void array(const Json &j, std::size_t max) {
  if (!j.is_array() || j.size() > max)
    invalid("array outside bounds");
}
Json seal(Json j) {
  j.erase("digest");
  j["digest"] = engine::tagged_sha256(j.dump());
  return j;
}
void sealed(const Json &j) {
  if (!j.is_object() || !j.contains("digest") || seal(j) != j)
    invalid("seal mismatch");
}
void hash(const Json &j) {
  if (!j.is_string())
    invalid("invalid digest");
  auto s = j.get<std::string>();
  if (s.size() != 71 || !s.starts_with("sha256:") ||
      !std::all_of(s.begin() + 7, s.end(), [](char c) {
        return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
      }))
    invalid("invalid digest");
}
std::int64_t number(const Json &j, std::int64_t low, std::int64_t high) {
  if (!j.is_number_integer() || j < low || j > high)
    invalid("integer outside bounds");
  return j.get<std::int64_t>();
}
void uri(const std::string &s) {
  auto offset = s.starts_with("https://")  ? 8u
                : s.starts_with("http://") ? 7u
                                           : 0u;
  if (!offset || s.size() > 4096 ||
      std::any_of(s.begin(), s.end(), [](unsigned char c) {
        return c <= 32 || c >= 127 || c == '\\' || c == '#' || c == '"' ||
               c == '<' || c == '>';
      }))
    invalid("unsupported URI profile");
  auto end = s.find_first_of("/?", offset);
  auto authority = s.substr(offset, end == s.npos ? s.npos : end - offset);
  auto colon = authority.find(':');
  auto host = authority.substr(0, colon);
  auto alnum = [](unsigned char c) {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
           (c >= '0' && c <= '9');
  };
  if (host.empty() || !alnum(host[0]) ||
      !std::all_of(host.begin(), host.end(), [&](unsigned char c) {
        return alnum(c) || c == '.' || c == '-';
      }))
    invalid("unsupported URI host");
  if (colon != authority.npos) {
    auto port = authority.substr(colon + 1);
    unsigned n = 0;
    auto [p, e] = std::from_chars(port.data(), port.data() + port.size(), n);
    if (e != std::errc{} || p != port.data() + port.size() || n < 1 ||
        n > 65535 || std::to_string(n) != port)
      invalid("unsupported URI port");
  }
}
void observed(const std::string &s) {
  if (s.size() != 20 || s[4] != '-' || s[7] != '-' || s[10] != 'T' ||
      s[13] != ':' || s[16] != ':' || s[19] != 'Z')
    invalid("invalid UTC observation time");
  for (std::size_t i = 0; i < s.size(); ++i)
    if (i != 4 && i != 7 && i != 10 && i != 13 && i != 16 && i != 19 &&
        (s[i] < '0' || s[i] > '9'))
      invalid("invalid UTC digits");
  int y = std::stoi(s.substr(0, 4)), m = std::stoi(s.substr(5, 2)),
      d = std::stoi(s.substr(8, 2));
  if (y < 1 ||
      !std::chrono::year_month_day{std::chrono::year{y},
                                   std::chrono::month{static_cast<unsigned>(m)},
                                   std::chrono::day{static_cast<unsigned>(d)}}
           .ok() ||
      std::stoi(s.substr(11, 2)) > 23 || std::stoi(s.substr(14, 2)) > 59 ||
      std::stoi(s.substr(17, 2)) > 59)
    invalid("invalid UTC calendar");
}
void definition(const Json &d) {
  fields(d, {"source_id", "publisher", "authority_role", "subject_ids",
             "locators"});
  id(d, "source_id");
  text(d, "publisher", 256);
  text(d, "authority_role", 256);
  array(d.at("subject_ids"), 32);
  array(d.at("locators"), 16);
  if (d.at("subject_ids").empty() || d.at("locators").empty())
    invalid("source requires subject scope and locators");
  std::set<std::string> subjects, locators, uris;
  for (const auto &s : d.at("subject_ids"))
    if (!subjects.insert(id(Json{{"id", s}}, "id")).second)
      invalid("duplicate subject");
  for (const auto &l : d.at("locators")) {
    fields(l, {"id", "uri", "format"});
    auto u = text(l, "uri");
    uri(u);
    auto f = text(l, "format", 16);
    if (!locators.insert(id(l, "id")).second || !uris.insert(u).second ||
        (f != "html" && f != "opaque"))
      invalid("duplicate locator or unsupported format");
  }
}
void source(const Json &s) {
  fields(s,
         {"protocol", "definition", "generation", "previous_digest", "digest"});
  sealed(s);
  if (s.at("protocol") != "symphony.shv.source-revision.v1")
    invalid("unsupported source protocol");
  definition(s.at("definition"));
  auto g = number(s.at("generation"), 1, 32);
  if (g == 1) {
    if (!s.at("previous_digest").is_null())
      invalid("first revision has predecessor");
  } else
    hash(s.at("previous_digest"));
}
Json plan(const Json &p) {
  fields(p, {"operation_id", "current", "desired", "reason"});
  id(p, "operation_id");
  text(p, "reason");
  definition(p.at("desired"));
  const auto &current = p.at("current");
  std::int64_t generation = 1;
  Json previous = nullptr;
  std::string kind = "onboard";
  if (!current.is_null()) {
    source(current);
    const auto &d = current.at("definition");
    if (d.at("source_id") != p.at("desired").at("source_id"))
      invalid("source identity cannot change within a chain");
    if (d == p.at("desired"))
      invalid("source plan makes no change");
    generation = number(current.at("generation"), 1, 31) + 1;
    previous = current.at("digest");
    kind = "relocation";
    for (auto key : {"publisher", "authority_role", "subject_ids"})
      if (d.at(key) != p.at("desired").at(key))
        kind = "authority_change";
  }
  auto next = seal(Json{{"protocol", "symphony.shv.source-revision.v1"},
                        {"definition", p.at("desired")},
                        {"generation", generation},
                        {"previous_digest", previous}});
  return seal(Json{{"protocol", "symphony.shv.source-plan.v1"},
                   {"operation_id", p.at("operation_id")},
                   {"expected_state_digest", previous},
                   {"change_kind", kind},
                   {"reason", p.at("reason")},
                   {"source", next}});
}
void check_deadline(const engine::Request &r) {
  if (engine::unix_time_ms() > r.deadline_unix_ms)
    throw engine::Error("request.deadline", "deadline exceeded", 4);
}
Json capture(const engine::Request &r, const Json &p) {
  fields(p, {"source_root", "source", "locator_id", "resolved_uri",
             "redirect_chain", "observed_at", "upstream_revision", "manifest",
             "completeness", "issues"});
  auto root = text(p, "source_root");
  auto rp = std::filesystem::path(root);
  if (!rp.is_absolute() || rp.lexically_normal().string() != root ||
      (root != "/" && root.ends_with('/')))
    invalid("source_root must be normalized absolute path");
  source(p.at("source"));
  auto locator = id(p, "locator_id");
  Json selected = nullptr;
  for (const auto &l : p.at("source").at("definition").at("locators"))
    if (l.at("id") == locator)
      selected = l;
  if (selected.is_null())
    invalid("locator outside exact source revision");
  auto resolved = text(p, "resolved_uri");
  uri(resolved);
  array(p.at("redirect_chain"), 8);
  std::string last = selected.at("uri").get<std::string>();
  std::set<std::string> hops{last};
  for (const auto &h : p.at("redirect_chain")) {
    auto u = text(Json{{"uri", h}}, "uri");
    uri(u);
    if (!hops.insert(u).second)
      invalid("redirect loop");
    last = u;
  }
  if (last != resolved)
    invalid("resolved URI differs from recorded route");
  observed(text(p, "observed_at", 20));
  if (!p.at("upstream_revision").is_null()) {
    fields(p.at("upstream_revision"), {"scheme", "value"});
    text(p.at("upstream_revision"), "scheme", 128);
    text(p.at("upstream_revision"), "value", 512);
  }
  auto completeness = text(p, "completeness", 16);
  array(p.at("issues"), 32);
  std::set<std::string> issues;
  for (const auto &i : p.at("issues"))
    if (!issues.insert(text(Json{{"issue", i}}, "issue", 512)).second)
      invalid("duplicate capture issue");
  if ((completeness != "complete" && completeness != "partial") ||
      (completeness == "partial" && issues.empty()))
    invalid("capture completeness requires explanation");
  const auto &m = p.at("manifest");
  fields(m, {"id", "path", "bytes", "digest", "format"});
  id(m, "id");
  hash(m.at("digest"));
  auto path = text(m, "path");
  if (!engine::is_safe_relative_path(path))
    invalid("unsafe source path");
  auto bytes = number(m.at("bytes"), 0, 1048576);
  if (m.at("format") != selected.at("format"))
    invalid("capture format differs from selected locator");
  check_deadline(r);
  auto raw = engine::read_regular_file_no_follow(root, path, 1048576,
                                                 r.deadline_unix_ms);
  if (static_cast<std::int64_t>(raw.size()) != bytes ||
      engine::tagged_sha256(raw) != m.at("digest").get<std::string>())
    invalid("capture bytes differ from manifest");
  auto result = p;
  result.erase("source_root");
  result["protocol"] = "symphony.shv.source-capture.v1";
  return seal(result);
}
void replay(const engine::Request &r, const Json &c, const std::string &root) {
  fields(c, {"protocol", "source", "locator_id", "resolved_uri",
             "redirect_chain", "observed_at", "upstream_revision", "manifest",
             "completeness", "issues", "digest"});
  sealed(c);
  if (c.at("protocol") != "symphony.shv.source-capture.v1")
    invalid("unsupported capture protocol");
  auto p = c;
  p.erase("protocol");
  p.erase("digest");
  p["source_root"] = root;
  if (capture(r, p) != c)
    invalid("capture replay mismatch");
}
Json project(const engine::Request &r, const Json &p) {
  fields(p, {"source_root", "captures"});
  auto root = text(p, "source_root");
  array(p.at("captures"), 8);
  // Even an empty graph requires a syntactically valid explicit root.
  auto rp = std::filesystem::path(root);
  if (!rp.is_absolute() || rp.lexically_normal().string() != root ||
      (root != "/" && root.ends_with('/')))
    invalid("invalid source_root");
  std::map<std::string, Json> nodes;
  Json edges = Json::array();
  std::set<std::string> ids, digests;
  std::int64_t total = 0;
  for (const auto &c : p.at("captures")) {
    total += number(c.at("manifest").at("bytes"), 0, 1048576);
    if (total > 4194304)
      invalid("cumulative capture byte bound");
  }
  for (const auto &c : p.at("captures")) {
    replay(r, c, root);
    auto cid = c.at("manifest").at("id").get<std::string>();
    if (!ids.insert(cid).second ||
        !digests.insert(c.at("digest").get<std::string>()).second)
      invalid("duplicate capture identity");
    auto s = c.at("source");
    auto sid = "source-revision:" + s.at("digest").get<std::string>().substr(7);
    auto nid = "capture:" + cid;
    nodes[sid] = Json{{"id", sid},
                      {"labels", Json::array({"source_revision"})},
                      {"properties", s}};
    nodes[nid] = Json{{"id", nid},
                      {"labels", Json::array({"source_capture"})},
                      {"properties", c}};
    edges.push_back(Json{
        {"id", "capture-source:" + cid},
        {"from", nid},
        {"to", sid},
        {"label", "captured_under"},
        {"properties", Json{{"source_id", s.at("definition").at("source_id")},
                            {"source_digest", s.at("digest")}}}});
  }
  Json rows = Json::array();
  for (const auto &[key, node] : nodes) {
    (void)key;
    rows.push_back(node);
  }
  std::sort(edges.begin(), edges.end(), [](const auto &a, const auto &b) {
    return a.at("id") < b.at("id");
  });
  auto bundle = seal(Json{{"protocol", "symphony.shv.source-bundle.v1"},
                          {"captures", p.at("captures")}});
  return seal(Json{{"protocol", "symphony.graph.exchange.v1"},
                   {"owner", Json{{"engine_id", engine_id},
                                  {"engine_version", version},
                                  {"artifact_protocol", bundle.at("protocol")},
                                  {"artifact_digest", bundle.at("digest")}}},
                   {"owner_artifact", bundle},
                   {"nodes", rows},
                   {"edges", edges}});
}
} // namespace
Json handle_request(const engine::Request &r) {
  check_deadline(r);
  const auto &p = r.payload;
  auto op = r.operation;
  if (op == "inspect") {
    fields(p, {});
    return descriptor();
  }
  if (op == "source_plan")
    return plan(p);
  if (op == "source_reduce") {
    fields(p, {"current", "plan"});
    const auto &v = p.at("plan");
    fields(v, {"protocol", "operation_id", "expected_state_digest",
               "change_kind", "reason", "source", "digest"});
    sealed(v);
    source(v.at("source"));
    auto expected = plan(Json{{"operation_id", v.at("operation_id")},
                              {"current", p.at("current")},
                              {"desired", v.at("source").at("definition")},
                              {"reason", v.at("reason")}});
    if (expected != v)
      invalid("plan differs from exact current-state transition");
    return seal(Json{{"protocol", "symphony.shv.source-transition.v1"},
                     {"operation_id", v.at("operation_id")},
                     {"expected_state_digest", v.at("expected_state_digest")},
                     {"source", v.at("source")}});
  }
  if (op == "source_status") {
    fields(p, {"history"});
    array(p.at("history"), 32);
    if (p.at("history").empty())
      invalid("history cannot be empty");
    Json hashes = Json::array(), previous = nullptr;
    for (const auto &s : p.at("history")) {
      source(s);
      if (previous.is_null()) {
        if (s.at("generation") != 1)
          invalid("history must start at generation one");
      } else {
        auto expected =
            plan(Json{{"operation_id", "history-check"},
                      {"current", previous},
                      {"desired", s.at("definition")},
                      {"reason", "Validate supplied source lineage"}});
        if (expected.at("source") != s)
          invalid("noncontiguous source history");
      }
      previous = s;
      hashes.push_back(s.at("digest"));
    }
    return seal(Json{{"protocol", "symphony.shv.source-status.v1"},
                     {"source", previous},
                     {"history_digests", hashes}});
  }
  if (op == "capture_import")
    return capture(r, p);
  if (op == "capture_compare") {
    fields(p, {"source_root", "previous", "current"});
    auto root = text(p, "source_root");
    const auto &a = p.at("previous"), &b = p.at("current");
    replay(r, a, root);
    replay(r, b, root);
    Json changes = Json::array();
    auto add = [&](bool changed, const char *label) {
      if (changed)
        changes.push_back(label);
    };
    bool same = a.at("source").at("definition").at("source_id") ==
                b.at("source").at("definition").at("source_id");
    add(!same, "source_identity");
    add(a.at("source").at("digest") != b.at("source").at("digest"),
        "source_revision");
    add(a.at("manifest").at("digest") != b.at("manifest").at("digest") ||
            a.at("manifest").at("bytes") != b.at("manifest").at("bytes"),
        "body");
    add(a.at("manifest").at("format") != b.at("manifest").at("format"),
        "representation");
    add(a.at("locator_id") != b.at("locator_id") ||
            a.at("resolved_uri") != b.at("resolved_uri") ||
            a.at("redirect_chain") != b.at("redirect_chain"),
        "acquisition_route");
    add(a.at("upstream_revision") != b.at("upstream_revision"),
        "upstream_revision");
    add(a.at("completeness") != b.at("completeness") ||
            a.at("issues") != b.at("issues"),
        "completeness");
    add(a.at("observed_at") != b.at("observed_at"), "observation_time");
    add(a.at("manifest").at("id") != b.at("manifest").at("id") ||
            a.at("manifest").at("path") != b.at("manifest").at("path"),
        "manifest_identity");
    return seal(Json{{"protocol", "symphony.shv.capture-comparison.v1"},
                     {"previous_digest", a.at("digest")},
                     {"current_digest", b.at("digest")},
                     {"same_logical_source", same},
                     {"changes", changes}});
  }
  if (op == "graph_project")
    return project(r, p);
  if (op == "graph_validate") {
    fields(p, {"source_root", "graph"});
    const auto &g = p.at("graph");
    fields(g,
           {"protocol", "owner", "owner_artifact", "nodes", "edges", "digest"});
    sealed(g);
    const auto &bundle = g.at("owner_artifact");
    fields(bundle, {"protocol", "captures", "digest"});
    sealed(bundle);
    if (bundle.at("protocol") != "symphony.shv.source-bundle.v1" ||
        project(r, Json{{"source_root", p.at("source_root")},
                        {"captures", bundle.at("captures")}}) != g)
      invalid("graph differs from source replay");
    return seal(Json{{"protocol", "symphony.shv.source-graph-validation.v1"},
                     {"graph_digest", g.at("digest")},
                     {"bundle_digest", bundle.at("digest")},
                     {"valid", true}});
  }
  invalid("unsupported operation");
}
} // namespace symphony::knowledge::shv_source
