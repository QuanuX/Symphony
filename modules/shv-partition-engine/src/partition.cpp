#include "partition.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <algorithm>
#include <set>
#include <string_view>
namespace symphony::knowledge::shv_partition {
namespace {
[[noreturn]] void bad() {
  throw engine::Error("shv-partition.invalid", "invalid partition contract", 4);
}
void fields(const Json &j, std::initializer_list<const char *> ks) {
  if (!j.is_object() || j.size() != ks.size())
    bad();
  for (auto k : ks)
    if (!j.contains(k))
      bad();
}
void array(const Json &j, std::size_t max) {
  if (!j.is_array() || j.size() > max)
    bad();
}
std::string token(const Json &v) {
  if (!v.is_string())
    bad();
  auto s = v.get<std::string>();
  if (s.empty() || s.size() > 128 ||
      !std::all_of(s.begin(), s.end(), [](unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
               (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.';
      }))
    bad();
  return s;
}
std::string hash(const Json &v) {
  if (!v.is_string())
    bad();
  auto s = v.get<std::string>();
  if (s.size() != 71 || !s.starts_with("sha256:") ||
      !std::all_of(s.begin() + 7, s.end(), [](char c) {
        return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
      }))
    bad();
  return s;
}
std::int64_t number(const Json &v, std::int64_t low, std::int64_t high) {
  if (!v.is_number_integer() || v < low || v > high)
    bad();
  return v.get<std::int64_t>();
}
Json seal(Json j) {
  j.erase("digest");
  j["digest"] = engine::tagged_sha256(j.dump());
  return j;
}
void engine_ref(const Json &v, bool source) {
  fields(v, {"engine_id", "version", "executable_digest"});
  hash(v["executable_digest"]);
  if (v["engine_id"] != (source ? "symphony-shv-source" : "symphony-shv"))
    bad();
  const bool extended = std::string_view(version) == "0.4.0-dev";
  if (v["version"] != "0.1.0-dev" &&
      !(extended && source && v["version"] == "0.2.0-dev") &&
      (source || (v["version"] != "0.2.0-dev" && v["version"] != "0.3.0-dev" &&
                  !(extended && v["version"] == "0.4.0-dev"))))
    bad();
}
Json partition(const Json &input) {
  fields(input, {"dependencies", "subject_ids"});
  auto d = input["dependencies"];
  fields(d, {"source_revision_digest", "source_engine", "kernel_engine",
             "captures", "mapping_digest", "catalogue_digest"});
  for (auto k :
       {"source_revision_digest", "mapping_digest", "catalogue_digest"})
    hash(d[k]);
  engine_ref(d["source_engine"], true);
  engine_ref(d["kernel_engine"], false);
  array(d["captures"], 8);
  if (d["captures"].empty())
    bad();
  std::set<std::string> ids, digests;
  std::int64_t bytes = 0;
  for (auto &c : d["captures"]) {
    fields(c, {"capture_id", "capture_digest", "content_digest", "bytes"});
    if (!ids.insert(token(c["capture_id"])).second ||
        !digests.insert(hash(c["capture_digest"])).second)
      bad();
    hash(c["content_digest"]);
    bytes += number(c["bytes"], 0, 1048576);
  }
  if (bytes > 4194304)
    bad();
  std::sort(d["captures"].begin(), d["captures"].end(),
            [](const Json &a, const Json &b) {
              return a["capture_id"] < b["capture_id"];
            });
  auto subjects = input["subject_ids"];
  array(subjects, 32);
  ids.clear();
  for (auto &v : subjects)
    if (!ids.insert(token(v)).second)
      bad();
  std::sort(subjects.begin(), subjects.end());
  return seal(Json{{"protocol", "symphony.shv.partition.v1"},
                   {"dependencies", d},
                   {"subject_ids", subjects}});
}
void validate_partition(const Json &p) {
  fields(p, {"protocol", "dependencies", "subject_ids", "digest"});
  if (partition(Json{{"dependencies", p["dependencies"]},
                     {"subject_ids", p["subject_ids"]}}) != p)
    bad();
}
void refs(const Json &r) {
  array(r, 128);
  std::set<std::string> seen;
  for (auto &v : r) {
    fields(v, {"partition_digest", "subject_id"});
    auto k = hash(v["partition_digest"]) + "/" + token(v["subject_id"]);
    if (!seen.insert(k).second)
      bad();
  }
}
std::string status(const Json &entries, const Json &r) {
  for (auto &e : entries) {
    if (e["partition_digest"] != r["partition_digest"])
      continue;
    if (e["partition"].is_null())
      return "missing_partition";
    for (auto &s : e["partition"]["subject_ids"])
      if (s == r["subject_id"])
        return "found";
    return "missing_subject";
  }
  return "unlisted_partition";
}
Json manifest(const Json &input) {
  fields(input, {"entries", "required_references"});
  const auto &entries = input["entries"];
  array(entries, 64);
  std::set<std::string> seen;
  std::size_t loaded = 0;
  for (auto &e : entries) {
    fields(e, {"partition_digest", "partition"});
    if (!seen.insert(hash(e["partition_digest"])).second)
      bad();
    if (!e["partition"].is_null()) {
      validate_partition(e["partition"]);
      if (e["partition_digest"] != e["partition"]["digest"])
        bad();
      ++loaded;
    }
  }
  refs(input["required_references"]);
  Json statuses = Json::array();
  for (auto &r : input["required_references"])
    statuses.push_back(Json{{"reference", r}, {"status", status(entries, r)}});
  return seal(Json{{"protocol", "symphony.shv.partition-manifest.v1"},
                   {"entries", entries},
                   {"required_references", input["required_references"]},
                   {"reference_statuses", statuses},
                   {"loaded_count", loaded},
                   {"missing_count", entries.size() - loaded},
                   {"complete_inventory", loaded == entries.size()}});
}
Json query(const Json &input) {
  fields(input, {"manifest", "selection", "limit", "cursor"});
  const auto &m = input["manifest"];
  fields(m, {"protocol", "entries", "required_references", "reference_statuses",
             "loaded_count", "missing_count", "complete_inventory", "digest"});
  if (manifest(Json{{"entries", m["entries"]},
                    {"required_references", m["required_references"]}}) != m)
    bad();
  const auto &selection = input["selection"];
  refs(selection);
  auto limit = number(input["limit"], 1, 32);
  auto sd = engine::tagged_sha256(Json{{"selection", selection}}.dump());
  std::size_t offset = 0;
  if (!input["cursor"].is_null()) {
    const auto &c = input["cursor"];
    fields(c, {"manifest_digest", "selection_digest", "offset"});
    if (c["manifest_digest"] != m["digest"] || c["selection_digest"] != sd)
      bad();
    offset = static_cast<std::size_t>(number(
        c["offset"], 1, static_cast<std::int64_t>(selection.size()) - 1));
  }
  auto end =
      std::min(selection.size(), offset + static_cast<std::size_t>(limit));
  Json rows = Json::array();
  for (auto i = offset; i < end; ++i)
    rows.push_back(Json{{"reference", selection[i]},
                        {"status", status(m["entries"], selection[i])}});
  Json next = nullptr;
  if (end < selection.size())
    next = Json{{"manifest_digest", m["digest"]},
                {"selection_digest", sd},
                {"offset", end}};
  return seal(Json{{"protocol", "symphony.shv.partition-query.v1"},
                   {"manifest_digest", m["digest"]},
                   {"selection_digest", sd},
                   {"offset", offset},
                   {"rows", rows},
                   {"total_selected", selection.size()},
                   {"next_cursor", next}});
}
} // namespace
Json handle_request(const engine::Request &r) {
  if (engine::unix_time_ms() > r.deadline_unix_ms)
    throw engine::Error("request.deadline", "deadline exceeded", 4);
  Json result;
  if (r.operation == "inspect") {
    fields(r.payload, {});
    result = descriptor();
  } else if (r.operation == "partition_build")
    result = partition(r.payload);
  else if (r.operation == "manifest_build")
    result = manifest(r.payload);
  else if (r.operation == "manifest_query")
    result = query(r.payload);
  else
    bad();
  if (engine::unix_time_ms() > r.deadline_unix_ms)
    throw engine::Error("request.deadline", "deadline exceeded", 4);
  return result;
}
} // namespace symphony::knowledge::shv_partition
