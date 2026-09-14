#include "publication.hpp"
#include "partition.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <algorithm>
#include <filesystem>
#include <regex>
#include <set>
#include <string_view>
static_assert(
    std::string_view(symphony::knowledge::shv_partition::version) ==
        "0.2.0-dev",
    "Review exact partition contract before upgrading publication dependency");
namespace symphony::knowledge::shv_publication {
namespace {
[[noreturn]] void invalid(const std::string &why) {
  throw engine::Error("shv-publication.invalid", why, 4);
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

void path(const Json &j, const char *key) {
  auto p = text(j, key);
  if (!p.starts_with('/') ||
      std::filesystem::path(p).lexically_normal().string() != p ||
      (p != "/" && p.ends_with('/')))
    invalid("clean absolute path required");
}
void tops(const Json &j, const char *key) {
  if (!std::regex_match(text(j, key, 36),
                        std::regex("[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-["
                                   "89ab][0-9a-f]{3}-[0-9a-f]{12}")))
    invalid("TOPS UUID required");
}
void installation(const Json &x, const std::string &module,
                  const std::string &engine,
                  const std::set<std::string> &versions) {
  fields(x, {"Role", "ModuleID", "EngineID", "Version", "Prefix", "ReceiptPath",
             "ReceiptDigest", "ReceiptProtocol", "ExecutablePath",
             "ExecutableDigest"});
  path(x, "Prefix");
  path(x, "ReceiptPath");
  path(x, "ExecutablePath");
  hash(x.at("ReceiptDigest"));
  hash(x.at("ExecutableDigest"));
  auto v = text(x, "Version", 128), p = text(x, "Prefix");
  if (x.at("Role") != module || x.at("ModuleID") != module ||
      x.at("EngineID") != engine || !versions.contains(v) ||
      x.at("ReceiptProtocol") != "symphony.knowledge.install-receipt.v2" ||
      x.at("ReceiptPath") != p + "/share/symphony/receipts/" + module + "/" +
                                 v + "/install-receipt.json" ||
      x.at("ExecutablePath") !=
          p + "/libexec/symphony/" + module + "/" + v + "/" + engine)
    invalid("installation identity differs");
}
void definition(const engine::Request &r, const Json &d) {
  fields(d, {"catalogue_id", "tops_id", "manifest", "policy",
             "partition_installation", "members"});
  id(d, "catalogue_id");
  tops(d, "tops_id");
  installation(d.at("partition_installation"), "shv-partition-engine",
               "symphony-shv-partition", {"0.2.0-dev"});
  const auto &m = d.at("manifest");
  auto req = r;
  req.operation = "manifest_build";
  req.payload = {{"entries", m.at("entries")},
                 {"required_references", m.at("required_references")}};
  if (shv_partition::handle_request(req) != m)
    invalid("manifest replay mismatch");
  array(m.at("entries"), 8);
  const auto &policy = d.at("policy");
  fields(policy, {"missing_partitions", "missing_references"});
  for (auto k : {"missing_partitions", "missing_references"})
    if (policy.at(k) != "allow" && policy.at(k) != "reject")
      invalid("explicit completeness policy required");
  if (policy.at("missing_partitions") == "reject" && m.at("missing_count") != 0)
    invalid("caller policy rejects missing partitions");
  if (policy.at("missing_references") == "reject")
    for (const auto &s : m.at("reference_statuses"))
      if (s.at("status") != "found")
        invalid("caller policy rejects unresolved references");
  array(d.at("members"), 8);
  std::set<std::string> loaded, seen;
  for (const auto &e : m.at("entries"))
    if (!e.at("partition").is_null())
      loaded.insert(e.at("partition_digest").get<std::string>());
  for (const auto &member : d.at("members")) {
    fields(member, {"partition_digest", "endpoint", "store", "bundle_digest",
                    "graph_digest", "source_installation",
                    "kernel_installation", "store_installation"});
    for (auto k : {"partition_digest", "bundle_digest", "graph_digest"})
      hash(member.at(k));
    auto pd = member.at("partition_digest").get<std::string>();
    if (!loaded.contains(pd) || !seen.insert(pd).second)
      invalid("member does not uniquely cover loaded partition");
    installation(member.at("source_installation"), "shv-source-engine",
                 "symphony-shv-source", {"0.1.0-dev"});
    installation(member.at("kernel_installation"), "shv-engine", "symphony-shv",
                 {"0.1.0-dev", "0.2.0-dev", "0.3.0-dev"});
    installation(member.at("store_installation"), "shv-graph-duckdb-connector",
                 "symphony-shv-graph-duckdb-connector",
                 {"0.1.0-dev", "0.2.0-dev", "0.3.0-dev"});
    const auto &ep = member.at("endpoint");
    fields(ep, {"bundle_path", "source_root", "state_root", "tops_id",
                "source_id", "source_prefix", "source_version", "kernel_prefix",
                "kernel_version"});
    for (auto k : {"bundle_path", "source_root", "state_root", "source_prefix",
                   "kernel_prefix"})
      path(ep, k);
    id(ep, "source_id");
    tops(ep, "tops_id");
    if (ep.at("tops_id") != d.at("tops_id") ||
        ep.at("source_prefix") !=
            member.at("source_installation").at("Prefix") ||
        ep.at("source_version") !=
            member.at("source_installation").at("Version") ||
        ep.at("kernel_prefix") !=
            member.at("kernel_installation").at("Prefix") ||
        ep.at("kernel_version") !=
            member.at("kernel_installation").at("Version"))
      invalid("endpoint owner differs");
    const auto &store = member.at("store");
    fields(store, {"root", "tops_id", "namespace", "snapshot_digest", "prefix",
                   "version"});
    path(store, "root");
    path(store, "prefix");
    tops(store, "tops_id");
    hash(store.at("snapshot_digest"));
    auto ns = text(store, "namespace", 128);
    if (!std::regex_match(ns,
                          std::regex("[A-Za-z0-9][A-Za-z0-9._:-]{0,127}")) ||
        store.at("tops_id") != d.at("tops_id") ||
        store.at("prefix") != member.at("store_installation").at("Prefix") ||
        store.at("version") != member.at("store_installation").at("Version"))
      invalid("store scope or installation differs");
    for (const auto &entry : m.at("entries"))
      if (entry.at("partition_digest") == pd) {
        const auto &dep = entry.at("partition").at("dependencies");
        for (auto pair : {std::pair{"source_engine", "source_installation"},
                          std::pair{"kernel_engine", "kernel_installation"}}) {
          const auto &x = member.at(pair.second);
          if (dep.at(pair.first) !=
              Json{{"engine_id", x.at("EngineID")},
                   {"version", x.at("Version")},
                   {"executable_digest", x.at("ExecutableDigest")}})
            invalid("partition owner dependency differs");
        }
      }
  }
  if (seen != loaded)
    invalid("all loaded partitions require replay and storage bindings");
}
void head(const engine::Request &r, const Json &h) {
  fields(h,
         {"protocol", "definition", "generation", "previous_digest", "digest"});
  sealed(h);
  if (h.at("protocol") != "symphony.shv.publication-head.v1")
    invalid("head protocol differs");
  definition(r, h.at("definition"));
  auto g = number(h.at("generation"), 1, 32);
  if (g == 1) {
    if (!h.at("previous_digest").is_null())
      invalid("initial predecessor differs");
  } else
    hash(h.at("previous_digest"));
}
Json plan(const engine::Request &r, const Json &p) {
  fields(p, {"operation_id", "current", "desired", "reason"});
  id(p, "operation_id");
  text(p, "reason");
  definition(r, p.at("desired"));
  auto current = p.at("current");
  std::int64_t generation = 1;
  Json previous = nullptr;
  if (!current.is_null()) {
    head(r, current);
    for (auto k : {"catalogue_id", "tops_id"})
      if (current.at("definition").at(k) != p.at("desired").at(k))
        invalid("catalogue scope cannot change");
    if (current.at("definition") == p.at("desired"))
      invalid("no publication change");
    generation = number(current.at("generation"), 1, 31) + 1;
    previous = current.at("digest");
  }
  return seal(
      Json{{"protocol", "symphony.shv.publication-plan.v1"},
           {"operation_id", p.at("operation_id")},
           {"expected_state_digest", previous},
           {"change_kind", "publish"},
           {"reason", p.at("reason")},
           {"head", seal(Json{{"protocol", "symphony.shv.publication-head.v1"},
                              {"definition", p.at("desired")},
                              {"generation", generation},
                              {"previous_digest", previous}})}});
}
} // namespace
Json handle_request(const engine::Request &r) {
  if (engine::unix_time_ms() >= r.deadline_unix_ms)
    invalid("deadline exceeded");
  const auto &p = r.payload;
  if (r.operation == "inspect") {
    fields(p, {});
    return descriptor();
  }
  if (r.operation == "publication_plan")
    return plan(r, p);
  if (r.operation == "publication_reduce") {
    fields(p, {"current", "plan"});
    const auto &x = p.at("plan");
    fields(x, {"protocol", "operation_id", "expected_state_digest",
               "change_kind", "reason", "head", "digest"});
    auto expected = plan(r, Json{{"current", p.at("current")},
                                 {"desired", x.at("head").at("definition")},
                                 {"operation_id", x.at("operation_id")},
                                 {"reason", x.at("reason")}});
    if (expected != x)
      invalid("plan or expected head differs");
    return seal(Json{{"protocol", "symphony.shv.publication-transition.v1"},
                     {"operation_id", x.at("operation_id")},
                     {"expected_state_digest", x.at("expected_state_digest")},
                     {"head", x.at("head")}});
  }
  if (r.operation == "publication_status") {
    fields(p, {"history"});
    array(p.at("history"), 32);
    if (p.at("history").empty())
      invalid("history empty");
    Json ds = Json::array(), previous = nullptr, scope = nullptr;
    std::int64_t generation = 0;
    for (const auto &h : p.at("history")) {
      head(r, h);
      if (h.at("generation") != ++generation ||
          h.at("previous_digest") != previous)
        invalid("history chain differs");
      Json key = {h.at("definition").at("tops_id"),
                  h.at("definition").at("catalogue_id")};
      if (!scope.is_null() && scope != key)
        invalid("history scope differs");
      scope = key;
      previous = h.at("digest");
      ds.push_back(previous);
    }
    return seal(Json{{"protocol", "symphony.shv.publication-status.v1"},
                     {"head", p.at("history").back()},
                     {"history_digests", ds}});
  }
  invalid("unsupported operation");
}
} // namespace symphony::knowledge::shv_publication
