#include "profile.hpp"
#include "shv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include <algorithm>
#include <filesystem>
#include <map>
#include <regex>
#include <set>
#include <string_view>
static_assert(std::string_view(symphony::knowledge::shv::version) ==
                  symphony::knowledge::shv_profile::embedded_kernel_version,
              "Review the exact kernel reader before changing admission");
namespace symphony::knowledge::shv_profile {
namespace k = symphony::knowledge::shv;
namespace {
std::string text(const Json &v, const char *key, std::size_t bound) {
  auto result = k::text(v, key, bound);
  if (std::any_of(result.begin(), result.end(),
                  [](unsigned char c) { return c < 32 || c == 127; }))
    k::invalid("control character in profile text");
  return result;
}
void object(const Json &v) {
  if (!v.is_object())
    k::invalid("expected an object");
}
void hash(const Json &v) {
  if (!v.is_string() || !std::regex_match(v.get<std::string>(),
                                          std::regex("sha256:[0-9a-f]{64}")))
    k::invalid("digest required");
}
void type(const Json &v) {
  if (v != "string" && v != "integer" && v != "date" && v != "tokens" &&
      v != "quarter_20yy" && v != "table_rows")
    k::invalid("unsupported typed metric");
}
void root(const Json &v) {
  if (!v.is_string())
    k::invalid("root must be text");
  auto s = text(Json{{"root", v}}, "root", 4096);
  if (s.empty() || s.size() > 4096 || s.find('\0') != s.npos ||
      (s.size() > 1 && s.back() == '/') ||
      !std::filesystem::path(s).is_absolute() ||
      std::filesystem::path(s).lexically_normal().string() != s)
    k::invalid("root must be clean absolute");
}
Json compile(const Json &d) {
  k::fields(d, {"id", "revision", "hardware_class", "metrics", "extensions"});
  k::ident(d, "id");
  k::ident(d, "revision");
  k::ident(d, "hardware_class");
  object(d.at("extensions"));
  k::array(d.at("metrics"), 16);
  std::set<std::string> seen;
  for (const auto &m : d.at("metrics")) {
    k::fields(m, {"predicate", "value_type", "qualifier", "required",
                  "description", "extensions"});
    if (!seen.insert(k::ident(m, "predicate")).second)
      k::invalid("duplicate metric");
    type(m.at("value_type"));
    text(m, "qualifier", 256);
    text(m, "description", 4096);
    object(m.at("extensions"));
    if (!m.at("required").is_boolean())
      k::invalid("required must be boolean");
  }
  return k::seal(
      Json{{"protocol", "symphony.shv.class-profile.v1"}, {"definition", d}});
}
void profile(const Json &p) {
  k::fields(p, {"protocol", "definition", "digest"});
  if (compile(p.at("definition")) != p)
    k::invalid("class profile differs");
}
void mappings(const Json &rows, bool portable) {
  k::array(rows, 32);
  std::set<std::string> ids;
  std::size_t count = 0;
  for (const auto &m : rows) {
    bool pdf = m.value("interpretation_profile", std::string{}) == "pdf_opn.v1";
    bool tables = m.contains("interpretation_profile") && !pdf;
    if (pdf) {
      k::fields(m, {"id", "manufacturer", "model", "hardware_class",
                    "source_id", "heading_section", "interpretation_profile",
                    "document", "fields"});
      const auto &d = m.at("document");
      if (portable || d.contains("decoder_binding")) {
        k::fields(d, {"decoder_binding", "extraction"});
        k::ident(d, "decoder_binding");
      } else {
        k::fields(d, {"decoder_root", "extraction"});
        root(d.at("decoder_root"));
      }
      if (m.at("heading_section") != "table8")
        k::invalid("PDF heading differs");
      object(d.at("extraction"));
    } else if (tables) {
      k::fields(m,
                {"id", "manufacturer", "model", "hardware_class", "source_id",
                 "heading_section", "interpretation_profile", "fields"});
      if (m.at("interpretation_profile") != "scoped_tables.v1")
        k::invalid("unknown interpretation profile");
    } else {
      k::fields(m, {"id", "manufacturer", "model", "hardware_class",
                    "source_id", "heading_section", "field_section", "fields"});
      text(m, "field_section", 128);
    }
    if (!ids.insert(k::ident(m, "id")).second)
      k::invalid("duplicate subject");
    k::ident(m, "source_id");
    k::ident(m, "hardware_class");
    text(m, "manufacturer", 256);
    text(m, "model", 256);
    text(m, "heading_section", 128);
    k::array(m.at("fields"), 16);
    count += m.at("fields").size();
    std::set<std::string> predicates;
    for (const auto &f : m.at("fields")) {
      if (!predicates.insert(k::ident(f, "predicate")).second)
        k::invalid("duplicate predicate");
      type(f.at("value_type"));
      text(f, "qualifier", 256);
      if (tables && f.at("value_type") == "table_rows") {
        k::fields(
            f, {"predicate", "section", "columns", "value_type", "qualifier"});
        k::array(f.at("columns"), 16);
        if (f.at("columns").empty())
          k::invalid("columns required");
        std::set<std::string> cols;
        for (const auto &c : f.at("columns")) {
          auto v = text(Json{{"v", c}}, "v", 256);
          if (!cols.insert(v).second)
            k::invalid("duplicate column");
        }
      } else {
        if (tables)
          k::fields(f, {"predicate", "section", "label", "next_label",
                        "value_type", "qualifier"});
        else
          k::fields(f, {"predicate", "label", "next_label", "value_type",
                        "qualifier"});
        text(f, "label", 256);
        if (!tables || !f.at("next_label").is_null())
          text(f, "next_label", 256);
      }
      if (tables)
        text(f, "section", 128);
      if (!tables && (f.at("value_type") == "quarter_20yy" ||
                      f.at("value_type") == "table_rows"))
        k::invalid("structured type requires scoped table profile");
      if (pdf && (f.at("label") != "OPN" || f.at("next_label") != "Model" ||
                  f.at("value_type") != "string" ||
                  f.at("qualifier") != "issuer=AMD;namespace=opn;profile=1"))
        k::invalid("PDF field profile differs");
      if (f.at("predicate") == "model_introduction" &&
          f.at("value_type") != "date" &&
          !(tables && f.at("value_type") == "quarter_20yy"))
        k::invalid("introduction must have explicit date precision");
    }
  }
  if (count > 256)
    k::invalid("assertion bound exceeded");
}
Json diagnose(const Json &p) {
  k::fields(p, {"profile", "mapping"});
  profile(p.at("profile"));
  mappings(p.at("mapping"), false);
  const auto &d = p.at("profile").at("definition");
  Json rows = Json::array(),
       counts = {{"conformant", 0}, {"incomplete", 0}, {"not_applicable", 0}};
  for (const auto &m : p.at("mapping")) {
    Json findings = Json::array(), extensions = Json::array();
    std::string status = "not_applicable";
    if (m.at("hardware_class") == d.at("hardware_class")) {
      status = "conformant";
      std::map<std::string, Json> actual;
      std::set<std::string> known;
      for (const auto &f : m.at("fields"))
        actual.emplace(k::ident(f, "predicate"), f);
      for (const auto &metric : d.at("metrics")) {
        auto pred = k::ident(metric, "predicate");
        known.insert(pred);
        auto i = actual.find(pred);
        Json problems = Json::array(), observed = nullptr;
        std::string state;
        if (i == actual.end())
          state = metric.at("required") == true ? "unmapped_required"
                                                : "unmapped_optional";
        else {
          observed = Json{{"value_type", i->second.at("value_type")},
                          {"qualifier", i->second.at("qualifier")}};
          for (const auto *key : {"value_type", "qualifier"})
            if (i->second.at(key) != metric.at(key))
              problems.push_back(key);
          state = problems.empty() ? "matched" : "mismatch";
        }
        if (state == "unmapped_required" || state == "mismatch")
          status = "incomplete";
        findings.push_back(Json{{"predicate", pred},
                                {"status", state},
                                {"expected", metric},
                                {"observed", observed},
                                {"differences", problems}});
      }
      for (const auto &[pred, f] : actual) {
        static_cast<void>(f);
        if (!known.contains(pred))
          extensions.push_back(pred);
      }
    }
    counts[status] = counts[status].get<int>() + 1;
    rows.push_back(Json{{"subject_id", m.at("id")},
                        {"status", status},
                        {"findings", findings},
                        {"extension_predicates", extensions}});
  }
  return k::seal(Json{{"protocol", "symphony.shv.mapping-diagnostics.v1"},
                      {"input", p},
                      {"subjects", rows},
                      {"counts", counts},
                      {"evidence_scope", "mapping_declarations_only"}});
}
std::set<std::string> source_ids(const Json &sources) {
  k::array(sources, 8);
  std::set<std::string> ids, paths;
  std::uint64_t total = 0;
  for (const auto &s : sources) {
    k::fields(s, {"id", "path", "bytes", "digest", "format"});
    if (!ids.insert(k::ident(s, "id")).second)
      k::invalid("duplicate source");
    auto path = text(s, "path", 4096);
    if (!engine::is_safe_relative_path(path) || !paths.insert(path).second)
      k::invalid("unsafe or duplicate source path");
    hash(s.at("digest"));
    if (!s.at("bytes").is_number_integer() || s.at("bytes") < 0 ||
        s.at("bytes") > 1048576)
      k::invalid("source byte bound");
    total += s.at("bytes").get<std::uint64_t>();
    if (s.at("format") != "html" && s.at("format") != "opaque")
      k::invalid("unsupported source format");
  }
  if (total > 4194304)
    k::invalid("aggregate source byte bound");
  return ids;
}
Json universe(const Json &p) {
  k::fields(p, {"id", "revision", "kernel_version", "coverage", "profiles",
                "sources", "mapping", "locators", "extensions"});
  k::ident(p, "id");
  k::ident(p, "revision");
  if (p.at("kernel_version") != k::version)
    k::invalid("unsupported exact kernel contract");
  object(p.at("extensions"));
  static_cast<void>(k::coverage_plan(
      Json{{"profile", p.at("coverage")}, {"subjects", Json::array()}}));
  mappings(p.at("mapping"), true);
  k::array(p.at("profiles"), 16);
  std::set<std::string> profiles;
  for (const auto &v : p.at("profiles")) {
    profile(v);
    if (!profiles.insert(k::ident(v.at("definition"), "id")).second)
      k::invalid("duplicate profile ID");
  }
  auto ids = source_ids(p.at("sources"));
  for (const auto &m : p.at("mapping"))
    if (!ids.contains(k::ident(m, "source_id")))
      k::invalid("mapping source absent");
  k::array(p.at("locators"), 8);
  std::set<std::string> locators;
  for (const auto &l : p.at("locators")) {
    k::fields(l, {"source_id", "uri", "upstream_revision"});
    auto id = k::ident(l, "source_id");
    if (!ids.contains(id) || !locators.insert(id).second)
      k::invalid("unknown or duplicate locator source");
    text(l, "uri", 4096);
    if (!l.at("upstream_revision").is_null())
      text(l, "upstream_revision", 256);
  }
  return k::seal(
      Json{{"protocol", "symphony.shv.universe.v1"}, {"definition", p}});
}
Json bind(const engine::Request &r, const Json &p) {
  k::fields(p, {"universe", "bindings"});
  const auto &u = p.at("universe");
  k::fields(u, {"protocol", "definition", "digest"});
  if (universe(u.at("definition")) != u)
    k::invalid("universe identity differs");
  const auto &d = u.at("definition"), &b = p.at("bindings");
  k::fields(b, {"source_root", "decoders"});
  root(b.at("source_root"));
  object(b.at("decoders"));
  std::set<std::string> used;
  auto mapping = d.at("mapping");
  for (auto &m : mapping)
    if (m.value("interpretation_profile", std::string{}) == "pdf_opn.v1") {
      auto &doc = m.at("document");
      auto key = k::ident(doc, "decoder_binding");
      if (!b.at("decoders").contains(key))
        k::invalid("decoder binding missing");
      used.insert(key);
      root(b.at("decoders").at(key));
      doc.erase("decoder_binding");
      doc["decoder_root"] = b.at("decoders").at(key);
    }
  if (used.size() != b.at("decoders").size())
    k::invalid("unused decoder binding");
  Json input = {{"source_root", b.at("source_root")},
                {"sources", d.at("sources")},
                {"subjects", mapping}};
  auto catalogue = k::catalogue_build(r, input);
  Json summaries = Json::array();
  for (auto s : catalogue.at("subjects")) {
    s.erase("assertions");
    summaries.push_back(s);
  }
  auto coverage = k::coverage_plan(
      Json{{"profile", d.at("coverage")}, {"subjects", summaries}});
  Json conformance = Json::array(), unprofiled = Json::array();
  std::set<std::string> classes, declared;
  for (const auto &pr : d.at("profiles")) {
    declared.insert(k::ident(pr.at("definition"), "hardware_class"));
    conformance.push_back(
        diagnose(Json{{"profile", pr}, {"mapping", mapping}}));
  }
  for (const auto &m : mapping)
    classes.insert(k::ident(m, "hardware_class"));
  for (const auto &c : classes)
    if (!declared.contains(c))
      unprofiled.push_back(c);
  return k::seal(Json{{"protocol", "symphony.shv.universe-binding.v1"},
                      {"input", p},
                      {"reader",
                       {{"engine_id", k::engine_id},
                        {"version", k::version},
                        {"mode", "compiled_exact_contract"}}},
                      {"catalogue_input", input},
                      {"catalogue", catalogue},
                      {"coverage", coverage},
                      {"conformance", conformance},
                      {"unprofiled_classes", unprofiled},
                      {"canonical_apply_enabled", false}});
}
} // namespace
void validate_mappings(const Json &rows, bool portable) {
  mappings(rows, portable);
}
void validate_sources(const Json &sources) {
  static_cast<void>(source_ids(sources));
}
Json handle_request(const engine::Request &r) {
  k::deadline(r);
  const auto &p = r.payload;
  if (r.operation == "extraction_diagnose")
    return extraction_diagnose(r);
  if (r.operation == "references_analyze")
    return references_analyze(p);
  if (r.operation == "inspect") {
    k::fields(p, {});
    return descriptor();
  }
  if (r.operation == "profile_compile")
    return compile(p);
  if (r.operation == "mapping_diagnose")
    return diagnose(p);
  if (r.operation == "universe_build")
    return universe(p);
  if (r.operation == "universe_bind")
    return bind(r, p);
  k::invalid("unsupported profile operation");
}
} // namespace symphony::knowledge::shv_profile
