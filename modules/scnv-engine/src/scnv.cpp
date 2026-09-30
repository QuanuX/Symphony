#include <symphony/snv/scnv.hpp>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <algorithm>
#include <map>
#include <set>
#include <string>
#include <utility>

namespace symphony::snv::scnv {
namespace {
namespace engine = symphony::knowledge::engine;
[[noreturn]] void reject(const char* code = "scnv.invalid_input",
                         const char* message = "SCNV input violates its exact contract",
                         int status = 2) {
  throw engine::Error(code, message, status);
}
void check(bool accepted) { if (!accepted) reject(); }
void fields(const Json& value, std::initializer_list<const char*> names) {
  check(value.is_object() && value.size() == names.size());
  for (const auto* name : names) check(value.contains(name));
}
// Validate UTF-8 without introducing Unicode normalization. This is also used
// by direct SDK calls, which do not necessarily pass through the JSON parser.
bool clean_utf8(std::string_view value) {
  std::size_t i = 0;
  while (i < value.size()) {
    const auto first = static_cast<unsigned char>(value[i]);
    if (first < 0x80) {
      if (first < 0x20 || first == 0x7f) return false;
      ++i; continue;
    }
    std::size_t width = 0;
    unsigned code = 0;
    if (first >= 0xc2 && first <= 0xdf) { width = 2; code = first & 0x1f; }
    else if (first >= 0xe0 && first <= 0xef) { width = 3; code = first & 0x0f; }
    else if (first >= 0xf0 && first <= 0xf4) { width = 4; code = first & 0x07; }
    else return false;
    if (width > value.size() - i) return false;
    for (std::size_t part = 1; part < width; ++part) {
      const auto next = static_cast<unsigned char>(value[i + part]);
      if ((next & 0xc0) != 0x80) return false;
      code = (code << 6) | (next & 0x3f);
    }
    if ((width == 2 && code < 0x80) || (width == 3 && code < 0x800) ||
        (width == 4 && code < 0x10000) || code > 0x10ffff ||
        (code >= 0xd800 && code <= 0xdfff) || (code >= 0x80 && code <= 0x9f))
      return false;
    i += width;
  }
  return true;
}
std::string text(const Json& value, const char* key, std::size_t max = 256) {
  check(value.is_object() && value.contains(key));
  const auto& item = value.at(key);
  check(item.is_string());
  const auto& string = item.get_ref<const std::string&>();
  check(!string.empty() && string.size() <= max && clean_utf8(string));
  return string;
}
std::string optional_text(const Json& value, const char* key, std::size_t max = 256) {
  check(value.is_object() && value.contains(key));
  if (value.at(key).is_null()) return {};
  return text(value, key, max);
}
std::int64_t integer(const Json& value, const char* key,
                     std::int64_t max = 9007199254740991LL) {
  check(value.is_object() && value.contains(key));
  const auto& item = value.at(key);
  check(item.is_number_integer());
  if (item.is_number_unsigned()) check(item.get<std::uint64_t>() <= static_cast<std::uint64_t>(max));
  const auto result = item.get<std::int64_t>();
  check(result >= 0 && result <= max);
  return result;
}
bool one_of(const std::string& value, std::initializer_list<const char*> choices) {
  return std::any_of(choices.begin(), choices.end(), [&](const auto* choice) { return value == choice; });
}
void deadline(std::int64_t end) {
  if (engine::unix_time_ms() >= end)
    reject("request.deadline_expired", "SCNV computation deadline expired", 3);
}
std::string subject_key(const Json& subject) {
  fields(subject, {"kind", "id"});
  const auto kind = text(subject, "kind", 32);
  check(one_of(kind, {"node", "incarnation", "resource", "cluster", "relationship"}));
  (void)text(subject, "id");
  return subject.dump();
}
bool supported_scope(const std::string& kind) {
  return one_of(kind, {"tops", "trog", "cluster"});
}
std::string scope_key(const Json& scope, bool require_supported = true) {
  fields(scope, {"kind", "id"});
  const auto kind = text(scope, "kind", 32);
  if (require_supported) check(supported_scope(kind));
  (void)text(scope, "id");
  return scope.dump();
}
std::size_t limit(const Json& limits) {
  fields(limits, {"max_records"});
  const auto bound = integer(limits, "max_records", 2048);
  check(bound == 512 || bound == 1024 || bound == 2048);
  return static_cast<std::size_t>(bound);
}
void record(const Json& r) {
  fields(r, {"record_id", "association_id", "kind", "previous_record_id", "origin_record_id",
             "subject", "name", "name_kind", "namespace", "source", "scope", "interval",
             "observed_unix_ms", "recorded_unix_ms"});
  (void)text(r, "record_id"); (void)text(r, "association_id");
  const auto kind = text(r, "kind", 32);
  check(one_of(kind, {"assignment", "correction", "retirement", "reuse", "restoration"}));
  const auto previous = optional_text(r, "previous_record_id");
  const auto origin = optional_text(r, "origin_record_id");
  check((kind == "correction" || kind == "retirement") == !previous.empty());
  check((kind == "reuse" || kind == "restoration") == !origin.empty());
  (void)subject_key(r.at("subject")); (void)scope_key(r.at("scope"));
  (void)text(r, "name", 512); (void)text(r, "namespace");
  check(one_of(text(r, "name_kind", 48), {"provider_resource_id", "provider_offering_name",
       "infrastructure_name", "node_identity", "cluster_name", "user_nickname", "short_name",
       "resource_name", "incarnation_name", "relationship_name", "external_name"}));
  const auto& source = r.at("source");
  fields(source, {"kind", "id"});
  check(one_of(text(source, "kind", 32), {"provider", "infrastructure", "symphony", "user", "observation"}));
  (void)text(source, "id");
  fields(r.at("interval"), {"from_unix_ms", "until_unix_ms"});
  const auto from = integer(r.at("interval"), "from_unix_ms");
  if (!r.at("interval").at("until_unix_ms").is_null())
    check(integer(r.at("interval"), "until_unix_ms") > from);
  if (kind == "retirement") check(!r.at("interval").at("until_unix_ms").is_null());
  (void)integer(r, "observed_unix_ms"); (void)integer(r, "recorded_unix_ms");
}
std::int64_t end_of(const Json& r) {
  const auto& value = r.at("interval").at("until_unix_ms");
  return value.is_null() ? 9007199254740992LL : value.get<std::int64_t>();
}
bool active_at(const Json& r, std::int64_t at) {
  return integer(r.at("interval"), "from_unix_ms") <= at && at < end_of(r);
}
struct Evidence {
  std::vector<const Json*> heads;
  std::map<std::string, const Json*> records;
  Json collisions = Json::array();
  Json subjects = Json::array();
  std::string digest;
};
Evidence evidence(const Json& records, const Json& limits, const std::string& coverage,
                  std::int64_t end) {
  const auto capacity = limit(limits);
  check(records.is_array());
  if (records.size() > capacity) reject("scnv.capacity_exceeded", "SCNV record capacity exceeded");
  check(one_of(coverage, {"complete", "partial"}));
  Evidence result;
  std::map<std::string, std::string> successors;
  std::map<std::string, std::vector<std::string>> associations;
  std::set<std::string> subject_ids;
  for (const auto& r : records) {
    deadline(end); record(r);
    const auto id = text(r, "record_id");
    check(result.records.emplace(id, &r).second);
    associations[text(r, "association_id")].push_back(id);
    subject_ids.insert(text(r.at("subject"), "id"));
  }
  for (const auto& [id, ptr] : result.records) {
    const auto& r = *ptr;
    const auto previous = optional_text(r, "previous_record_id");
    const auto origin = optional_text(r, "origin_record_id");
    if (!previous.empty()) {
      check(previous != id && result.records.contains(previous));
      const auto& p = *result.records.at(previous);
      check(r.at("association_id") == p.at("association_id"));
      if (!successors.emplace(previous, id).second)
        reject("scnv.lineage_conflict", "SCNV association history contains competing revisions");
      if (r.at("kind") == "retirement") {
        for (const auto* stable : {"subject", "name", "name_kind", "namespace", "source", "scope"})
          check(r.at(stable) == p.at(stable));
        check(r.at("interval").at("from_unix_ms") == p.at("interval").at("from_unix_ms"));
        check(end_of(r) <= end_of(p));
      }
    }
    if (!origin.empty()) {
      check(origin != id && result.records.contains(origin));
      const auto& p = *result.records.at(origin);
      check(r.at("association_id") != p.at("association_id"));
      check(r.at("name") == p.at("name") && r.at("scope") == p.at("scope"));
      check(!p.at("interval").at("until_unix_ms").is_null());
      check(integer(r.at("interval"), "from_unix_ms") >= end_of(p));
      if (r.at("kind") == "restoration") check(r.at("subject") == p.at("subject"));
    }
  }
  for (const auto& [association, ids] : associations) {
    (void)association;
    std::vector<std::string> roots;
    for (const auto& id : ids)
      if (result.records.at(id)->at("previous_record_id").is_null()) roots.push_back(id);
    check(roots.size() == 1);
    std::set<std::string> visited;
    auto current = roots.front();
    while (true) {
      deadline(end); check(visited.insert(current).second);
      if (!successors.contains(current)) break;
      current = successors.at(current);
    }
    check(visited.size() == ids.size());
    result.heads.push_back(result.records.at(current));
  }
  // Reuse must refer to a terminal retired revision, never a stale revision
  // which another explicit correction has already superseded.
  for (const auto& [id, ptr] : result.records) {
    (void)id;
    const auto origin = optional_text(*ptr, "origin_record_id");
    if (!origin.empty()) check(!successors.contains(origin));
  }
  std::sort(result.heads.begin(), result.heads.end(), [](const Json* left, const Json* right) {
    return left->at("record_id").get<std::string>() < right->at("record_id").get<std::string>();
  });
  std::map<std::string, std::vector<const Json*>> names;
  for (const auto* r : result.heads) {
    const auto key = Json{{"scope", r->at("scope")}, {"name", r->at("name")}}.dump();
    names[key].push_back(r);
  }
  for (const auto& [key, group] : names) {
    std::set<std::string> conflicting_records;
    std::map<std::string, Json> conflicting_subjects;
    for (std::size_t i = 0; i < group.size(); ++i) {
      deadline(end);
      for (std::size_t j = i + 1; j < group.size(); ++j) {
        const auto& a = *group[i]; const auto& b = *group[j];
        if (a.at("subject") == b.at("subject")) continue;
        const auto from = std::max(integer(a.at("interval"), "from_unix_ms"),
                                   integer(b.at("interval"), "from_unix_ms"));
        if (from < std::min(end_of(a), end_of(b))) {
          conflicting_records.insert(text(a, "record_id"));
          conflicting_records.insert(text(b, "record_id"));
          conflicting_subjects.emplace(a.at("subject").dump(), a.at("subject"));
          conflicting_subjects.emplace(b.at("subject").dump(), b.at("subject"));
        }
      }
    }
    if (!conflicting_records.empty()) {
      Json subjects = Json::array();
      for (const auto& [subject, item] : conflicting_subjects) { (void)subject; subjects.push_back(item); }
      const auto grouping = Json::parse(key);
      result.collisions.push_back({{"scope", grouping.at("scope")}, {"name", grouping.at("name")},
        {"kind", "unqualified_temporal_collision"}, {"record_ids", conflicting_records}, {"subjects", subjects}});
    }
  }
  for (const auto& subject : subject_ids) result.subjects.push_back(subject);
  result.digest = engine::tagged_sha256(Json{{"protocol", "scnv.evidence.v1"}, {"records", records},
                                          {"limits", limits}, {"coverage", coverage}}.dump());
  return result;
}
Json base(const Json& p, const Evidence& e) {
  return {{"owner", "scnv"}, {"owner_version", version},
          {"source_digest", engine::tagged_sha256(p.dump())}, {"subject_ids", e.subjects},
          {"evidence_digest", e.digest}, {"comparison", "utf8_bytes"}};
}
Json validate(const Json& p, std::int64_t end) {
  check(text(p, "protocol", 64) == "scnv.names.v1");
  const auto mode = text(p, "mode", 32);
  if (mode == "records") {
    fields(p, {"protocol", "mode", "records", "coverage", "limits"});
    const auto e = evidence(p.at("records"), p.at("limits"), text(p, "coverage", 16), end);
    auto out = base(p, e);
    out["protocol"] = "scnv.names-result.v1";
    out["mode"] = mode; out["valid"] = e.collisions.empty();
    out["collisions"] = e.collisions; out["record_count"] = p.at("records").size();
    out["association_count"] = e.heads.size(); out["coverage"] = p.at("coverage");
    return out;
  }
  check(mode == "transition");
  fields(p, {"protocol", "mode", "records", "coverage", "limits", "policy", "intent"});
  const auto& policy = p.at("policy");
  fields(policy, {"id", "version", "comparison", "scope", "collision", "interval"});
  (void)text(policy, "id"); (void)text(policy, "version", 64);
  check(text(policy, "comparison", 32) == "utf8_bytes" &&
        text(policy, "scope", 32) == "exact_scope" &&
        text(policy, "collision", 32) == "report" &&
        text(policy, "interval", 32) == "half_open");
  const auto& intent = p.at("intent");
  fields(intent, {"action", "expected_evidence_digest", "new_record", "reason"});
  const auto action = text(intent, "action", 32);
  check(one_of(action, {"assign", "correct", "retire", "reuse", "restore"}));
  (void)text(intent, "reason", 1024);
  const auto e = evidence(p.at("records"), p.at("limits"), text(p, "coverage", 16), end);
  if (text(intent, "expected_evidence_digest", 71) != e.digest)
    reject("scnv.expected_evidence_conflict", "SCNV predecessor evidence differs from the requested revision", 4);
  const auto& proposed = intent.at("new_record");
  record(proposed);
  const std::map<std::string, std::string> kinds{{"assign", "assignment"}, {"correct", "correction"},
     {"retire", "retirement"}, {"reuse", "reuse"}, {"restore", "restoration"}};
  check(proposed.at("kind") == kinds.at(action));
  Json records = p.at("records"); records.push_back(proposed);
  const auto candidate = evidence(records, p.at("limits"), text(p, "coverage", 16), end);
  auto out = base(p, candidate);
  out["protocol"] = "scnv.names-result.v1"; out["mode"] = mode;
  out["valid"] = candidate.collisions.empty(); out["collisions"] = candidate.collisions;
  out["record_count"] = records.size(); out["association_count"] = candidate.heads.size();
  out["coverage"] = p.at("coverage"); out["predecessor_digest"] = e.digest;
  out["intent_digest"] = engine::tagged_sha256(intent.dump());
  out["policy_digest"] = engine::tagged_sha256(policy.dump());
  out["proposed_records"] = Json::array({proposed});
  out["transition_kind"] = action;
  return out;
}
Json candidate(const Json& r) {
  return {{"subject", r.at("subject")}, {"association_id", r.at("association_id")},
    {"record_id", r.at("record_id")}, {"name", r.at("name")}, {"name_kind", r.at("name_kind")},
    {"namespace", r.at("namespace")}, {"source", r.at("source")}, {"scope", r.at("scope")},
    {"interval", r.at("interval")}};
}
Json resolve(const Json& p, std::int64_t end) {
  fields(p, {"protocol", "records", "coverage", "limits", "query"});
  check(text(p, "protocol", 64) == "scnv.resolve.v1");
  const auto e = evidence(p.at("records"), p.at("limits"), text(p, "coverage", 16), end);
  const auto& q = p.at("query");
  fields(q, {"mode", "name", "scope", "namespace", "name_kind", "as_of_unix_ms",
             "offset", "limit", "expected_evidence_digest"});
  const auto mode = text(q, "mode", 32);
  check(one_of(mode, {"exact", "scoped_candidates"}));
  const auto name = optional_text(q, "name", 512);
  if (mode == "exact") check(!name.empty());
  const auto namespace_filter = optional_text(q, "namespace");
  const auto kind_filter = optional_text(q, "name_kind", 48);
  if (!kind_filter.empty()) check(one_of(kind_filter, {"provider_resource_id", "provider_offering_name",
    "infrastructure_name", "node_identity", "cluster_name", "user_nickname", "short_name",
    "resource_name", "incarnation_name", "relationship_name", "external_name"}));
  const auto at = integer(q, "as_of_unix_ms");
  const auto scope = scope_key(q.at("scope"), false);
  const auto offset = static_cast<std::size_t>(integer(q, "offset", 2048));
  const auto page_limit = static_cast<std::size_t>(integer(q, "limit", 128));
  check(page_limit >= 1);
  const auto expected = optional_text(q, "expected_evidence_digest", 71);
  if (offset > 0 && expected.empty()) reject("scnv.snapshot_required", "SCNV continuation requires its exact evidence digest");
  if (!expected.empty() && expected != e.digest)
    reject("scnv.expected_evidence_conflict", "SCNV resolution snapshot differs from the requested evidence", 4);
  if (mode == "exact") check(offset == 0);
  auto out = base(p, e);
  out["protocol"] = "scnv.resolve-result.v1"; out["mode"] = mode;
  out["query"] = q; out["coverage"] = p.at("coverage");
  out["resolved_subject"] = nullptr; out["resolution_binding"] = nullptr;
  out["candidates"] = Json::array(); out["match_count"] = 0;
  out["next_offset"] = nullptr;
  if (!supported_scope(text(q.at("scope"), "kind", 32))) {
    out["status"] = "unsupported_scope"; return out;
  }
  std::vector<const Json*> matches;
  std::map<std::string, Json> subjects;
  for (const auto* r : e.heads) {
    deadline(end);
    if (scope_key(r->at("scope")) != scope || !active_at(*r, at) ||
        (!name.empty() && r->at("name") != name) ||
        (!namespace_filter.empty() && r->at("namespace") != namespace_filter) ||
        (!kind_filter.empty() && r->at("name_kind") != kind_filter)) continue;
    matches.push_back(r); subjects.emplace(r->at("subject").dump(), r->at("subject"));
  }
  out["match_count"] = matches.size();
  if (mode == "scoped_candidates") {
    check(offset <= matches.size());
    const auto until = std::min(matches.size(), offset + page_limit);
    for (std::size_t i = offset; i < until; ++i) out["candidates"].push_back(candidate(*matches[i]));
    if (until < matches.size()) out["next_offset"] = until;
    out["status"] = "candidates";
  } else {
    for (const auto* r : matches) out["candidates"].push_back(candidate(*r));
    if (subjects.size() > 1) out["status"] = "ambiguous";
    else if (p.at("coverage") == "partial") out["status"] = "insufficient";
    else if (subjects.empty()) out["status"] = "absent";
    else {
      out["status"] = "unique"; out["resolved_subject"] = subjects.begin()->second;
      Json record_ids = Json::array();
      for (const auto* r : matches) record_ids.push_back(r->at("record_id"));
      Json binding{{"protocol", "scnv.resolution-binding.v1"}, {"owner", "scnv"},
        {"owner_version", version}, {"subject", out.at("resolved_subject")},
        {"scope", q.at("scope")}, {"as_of_unix_ms", at}, {"evidence_digest", e.digest},
        {"record_ids", record_ids}};
      binding["binding_digest"] = engine::tagged_sha256(binding.dump());
      out["resolution_binding"] = binding;
    }
  }
  return out;
}
}

std::vector<symphony::snv::Operation> operations() {
  return {{"names_validate", "scnv.names.v1", "scnv.names-result.v1", {"inspect", "validate", "propose"}},
          {"names_resolve", "scnv.resolve.v1", "scnv.resolve-result.v1", {"inspect", "query"}}};
}

Json handle(std::string_view operation, const Json& payload, std::int64_t deadline_unix_ms) {
  deadline(deadline_unix_ms);
  if (operation == "names_validate") return validate(payload, deadline_unix_ms);
  if (operation == "names_resolve") return resolve(payload, deadline_unix_ms);
  reject("operation.unsupported", "Unsupported SCNV operation");
}
}
